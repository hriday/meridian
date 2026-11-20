package netdiag

import (
	"context"
	"net/netip"
	"strings"
	"sync"
	"testing"
)

// fakeEnv stubs every probe seam and restores them on cleanup.
func fakeEnv(t *testing.T, reach map[string]bool, neigh []Neighbor, osb OSBlockState, internet bool) {
	t.Helper()
	oi, orc, osi, on, oo, orh := probeIdentity, probeReach, sampleInternet, neighborsFn, osBlockFn, resolveHost
	t.Cleanup(func() {
		probeIdentity, probeReach, sampleInternet, neighborsFn, osBlockFn, resolveHost = oi, orc, osi, on, oo, orh
	})
	probeIdentity = func() (string, string, string, string, bool, bool, bool, error) {
		return "192.168.1.5", "192.168.1.1", "en0", "24", false, false, true, nil
	}
	probeReach = func(_ context.Context, h string) (bool, bool) { return reach[h], true }
	sampleInternet = func(context.Context) (bool, bool) { return internet, false }
	neighborsFn = func(context.Context) ([]Neighbor, error) { return neigh, nil }
	osBlockFn = func(context.Context, []string) (OSBlockState, string) { return osb, "fake" }
	resolveHost = func(context.Context, string) (string, error) { return "", context.Canceled }
}

var isoNeigh = []Neighbor{
	{IP: "192.168.1.1", State: "reachable"}, // the gateway, must be excluded
	{IP: "192.168.1.20", State: "reachable"},
	{IP: "192.168.1.21", State: "stale"},
}

func TestRun_HotspotIsolationPath(t *testing.T) {
	fakeEnv(t, map[string]bool{"192.168.1.1": true}, isoNeigh, OSBlockNotBlocked, true)
	r, err := Run(context.Background(), Options{}, nil)
	if err != nil || r.Verdict != VerdictClientIsolation {
		t.Fatalf("got %v err=%v, want client_isolation", r.Verdict, err)
	}
}

func TestRun_MacOSBlockPath(t *testing.T) {
	fakeEnv(t, map[string]bool{"192.168.1.1": true}, isoNeigh, OSBlockBlocked, true)
	r, _ := Run(context.Background(), Options{}, nil)
	if r.Verdict != VerdictOSBlockingMeridian {
		t.Fatalf("got %v, want os_blocking_meridian", r.Verdict)
	}
}

// When the gateway probe fails because the OS is blocking Meridian (not
// because the router is down), a positive OS-block signature against the
// gateway must flip the verdict to OS_BLOCKING_MERIDIAN, not GATEWAY_UNREACHABLE.
func TestRun_GatewayBlockedByOS(t *testing.T) {
	fakeEnv(t, map[string]bool{}, nil, OSBlockBlocked, true) // gateway not in reach map => fails
	r, _ := Run(context.Background(), Options{}, nil)
	if r.Verdict != VerdictOSBlockingMeridian {
		t.Fatalf("got %v, want os_blocking_meridian", r.Verdict)
	}
}

// A genuinely dead gateway (no positive OS-block signature) stays GATEWAY_UNREACHABLE.
func TestRun_GatewayDownStaysUnreachable(t *testing.T) {
	fakeEnv(t, map[string]bool{}, nil, OSBlockCouldNotDetermine, true)
	r, _ := Run(context.Background(), Options{}, nil)
	if r.Verdict != VerdictGatewayUnreachable {
		t.Fatalf("got %v, want gateway_unreachable", r.Verdict)
	}
}

func TestRun_GatewayExcludedFromPeers(t *testing.T) {
	var mu sync.Mutex
	var probed []string
	fakeEnv(t, map[string]bool{"192.168.1.1": true},
		[]Neighbor{{IP: "192.168.1.1", State: "reachable"}}, OSBlockNotBlocked, true)
	inner := probeReach
	probeReach = func(c context.Context, h string) (bool, bool) {
		mu.Lock()
		probed = append(probed, h)
		mu.Unlock()
		return inner(c, h)
	}
	r, _ := Run(context.Background(), Options{}, nil)
	if r.Verdict == VerdictAllClear || r.Verdict == VerdictClientIsolation {
		t.Fatalf("gateway-only sample must not be confident, got %v", r.Verdict)
	}
	if r.Verdict != VerdictInconclusive {
		t.Fatalf("got %v, want inconclusive", r.Verdict)
	}
	n := 0
	for _, h := range probed {
		if h == "192.168.1.1" {
			n++
		}
	}
	if n != 1 { // probed once as the gateway step only, never as a peer
		t.Fatalf("gateway probed %d times, want 1 (%v)", n, probed)
	}
}

func TestRun_StreamsStepsInOrder(t *testing.T) {
	fakeEnv(t, map[string]bool{"192.168.1.1": true, "192.168.1.20": true}, isoNeigh, OSBlockNotBlocked, true)
	var names []string
	r, _ := Run(context.Background(), Options{}, func(s StepResult) { names = append(names, s.Name) })
	want := []string{"identity", "gateway", "internet", "peers"}
	if len(names) != len(want) {
		t.Fatalf("steps %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("steps %v, want %v", names, want)
		}
	}
	if len(r.Steps) != len(names) || r.Verdict != VerdictAllClear {
		t.Fatalf("report steps=%d verdict=%v", len(r.Steps), r.Verdict)
	}
}

func TestRun_HostileTargetsNeverReachOSBlock(t *testing.T) {
	for _, tgt := range []string{"x; calc.exe", "fe80::1%x;calc.exe", "2001:db8::1", "fe80::1%en0"} {
		fakeEnv(t, map[string]bool{"192.168.1.1": true}, isoNeigh, OSBlockNotBlocked, true)
		var got []string
		osBlockFn = func(_ context.Context, p []string) (OSBlockState, string) { got = p; return OSBlockNotBlocked, "" }
		r, _ := Run(context.Background(), Options{Target: tgt}, nil)
		if len(got) == 0 {
			t.Fatalf("%q: osBlockFn never ran, test would be vacuous", tgt)
		}
		for _, p := range got {
			if a, err := netip.ParseAddr(p); err != nil || !a.Is4() || a.Zone() != "" {
				t.Fatalf("%q: non-IPv4 peer %q reached osBlockFn", tgt, p)
			}
		}
		if r.Verdict != VerdictInconclusive {
			t.Fatalf("%q: got %v, want inconclusive", tgt, r.Verdict)
		}
	}
}

func TestRun_ReachableTargetNotHostDown(t *testing.T) {
	fakeEnv(t, map[string]bool{"192.168.1.1": true, "192.168.1.50": true}, isoNeigh, OSBlockNotBlocked, true)
	r, _ := Run(context.Background(), Options{Target: "192.168.1.50"}, nil)
	if r.Verdict == VerdictHostDown || r.Verdict != VerdictAllClear {
		t.Fatalf("got %v, want all_clear", r.Verdict)
	}
}

func TestRun_TargetBlockedNoNeighbors_OSBlocking(t *testing.T) {
	fakeEnv(t, map[string]bool{"192.168.1.1": true}, nil, OSBlockBlocked, true)
	r, _ := Run(context.Background(), Options{Target: "192.168.1.50"}, nil)
	if r.Verdict != VerdictOSBlockingMeridian {
		t.Fatalf("got %v, want os_blocking_meridian", r.Verdict)
	}
}

func TestRun_ThinSampleNotConfident_Inconclusive(t *testing.T) {
	for _, st := range []OSBlockState{OSBlockNotBlocked, OSBlockCouldNotDetermine} {
		fakeEnv(t, map[string]bool{"192.168.1.1": true}, nil, st, true)
		r, _ := Run(context.Background(), Options{Target: "192.168.1.50"}, nil)
		if r.Verdict != VerdictInconclusive {
			t.Fatalf("OSBlock=%d: got %v, want inconclusive", st, r.Verdict)
		}
	}
}

func TestRun_ConfidentIsolationSoftCopy(t *testing.T) {
	fakeEnv(t, map[string]bool{"192.168.1.1": true}, isoNeigh, OSBlockNotBlocked, true)
	r, _ := Run(context.Background(), Options{}, nil)
	if r.Verdict != VerdictClientIsolation {
		t.Fatalf("got %v, want client_isolation", r.Verdict)
	}
	all := strings.ToLower(strings.Join(append(r.Causes, r.Statement), " "))
	if !strings.Contains(all, "powered off") || strings.Contains(all, "the network isolates clients") {
		t.Fatalf("copy not softened: %q", all)
	}
}
