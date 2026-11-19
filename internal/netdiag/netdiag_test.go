package netdiag

import (
	"strings"
	"testing"
)

// healthy returns observations for a fully healthy network.
func healthy() Observations {
	return Observations{
		GOOS: "linux", HasGateway: true, GatewayReachable: true,
		InternetReachable: true, PeerSampleConfident: true,
		PeersTried: 3, PeersReached: 3,
	}
}

func TestClassify_Table(t *testing.T) {
	tests := []struct {
		name string
		mod  func(o *Observations)
		want Verdict
	}{
		{"1 no gateway", func(o *Observations) { o.HasGateway = false }, VerdictDHCPFailed},
		{"1 link local", func(o *Observations) { o.IsLinkLocal = true }, VerdictDHCPFailed},
		{"2 gateway unreachable", func(o *Observations) { o.GatewayReachable = false }, VerdictGatewayUnreachable},
		{"3 captive", func(o *Observations) { o.InternetReachable = false; o.InternetIntercepted = true }, VerdictCaptivePortal},
		{"4 internet down", func(o *Observations) { o.InternetReachable = false }, VerdictInternetDown},
		{"5 os blocking", func(o *Observations) {
			o.PeersReached = 0
			o.OSBlock = OSBlockBlocked
		}, VerdictOSBlockingMeridian},
		{"6 isolation", func(o *Observations) {
			o.PeersReached = 0
			o.OSBlock = OSBlockNotBlocked
		}, VerdictClientIsolation},
		{"7 local access blocked", func(o *Observations) {
			o.PeersReached = 0
			o.OSBlock = OSBlockCouldNotDetermine
		}, VerdictLocalAccessBlocked},
		{"8 host down", func(o *Observations) { o.TargetGiven = true }, VerdictHostDown},
		{"9 all clear", func(o *Observations) {}, VerdictAllClear},
		{"9 all clear with target", func(o *Observations) { o.TargetGiven = true; o.TargetReached = true }, VerdictAllClear},
		{"10 probe error", func(o *Observations) { o.ProbeError = "arp scan failed" }, VerdictInconclusive},
		{"10 thin peer sample", func(o *Observations) {
			o.PeersReached = 0
			o.PeerSampleConfident = false
			o.OSBlock = OSBlockNotBlocked
		}, VerdictInconclusive},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := healthy()
			tt.mod(&o)
			if got := Classify(o); got != tt.want {
				t.Errorf("Classify = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassify_IPv6OnlyIsInconclusive(t *testing.T) {
	o := Observations{IPv6Only: true, HasGateway: false}
	if got := Classify(o); got != VerdictInconclusive {
		t.Errorf("got %q, want inconclusive (not dhcp_failed)", got)
	}
}

func TestClassify_ICMPUnavailableNotHostDown(t *testing.T) {
	o := healthy()
	o.TargetGiven = true
	o.ProbeError = "icmp unavailable: raw socket denied"
	if got := Classify(o); got == VerdictHostDown {
		t.Errorf("ICMP-unavailable must not read as host_down, got %q", got)
	}
	if got := Classify(o); got != VerdictInconclusive {
		t.Errorf("got %q, want inconclusive", got)
	}
}

func TestClassify_UnconfidentPeerSampleIsInconclusive(t *testing.T) {
	o := healthy()
	o.PeersTried, o.PeersReached = 1, 0
	o.PeerSampleConfident = false
	for _, s := range []OSBlockState{OSBlockUnknown, OSBlockBlocked, OSBlockNotBlocked, OSBlockCouldNotDetermine} {
		o.OSBlock = s
		if got := Classify(o); got != VerdictInconclusive {
			t.Errorf("OSBlock=%d: got %q, want inconclusive", s, got)
		}
	}
}

func TestClassify_CaptivePortal(t *testing.T) {
	o := healthy()
	o.InternetReachable, o.InternetIntercepted = false, true
	o.PeersReached = 0
	if got := Classify(o); got != VerdictCaptivePortal {
		t.Errorf("got %q", got)
	}
}

func TestClassify_InternetDownLANFine(t *testing.T) {
	o := healthy()
	o.InternetReachable = false
	if got := Classify(o); got != VerdictInternetDown {
		t.Errorf("got %q", got)
	}
}

func TestClassify_TargetDownButPeersReached(t *testing.T) {
	o := healthy()
	o.TargetGiven, o.TargetReached = true, false
	if got := Classify(o); got != VerdictHostDown {
		t.Errorf("got %q", got)
	}
}

func TestClassify_LocalAccessBlockedWhenOSBlockCouldNotDetermine(t *testing.T) {
	o := healthy()
	o.PeersReached = 0
	o.OSBlock = OSBlockCouldNotDetermine
	if got := Classify(o); got != VerdictLocalAccessBlocked {
		t.Errorf("got %q", got)
	}
}

func TestClassify_OSBlockingWhenBlocked(t *testing.T) {
	o := healthy()
	o.PeersReached = 0
	o.OSBlock = OSBlockBlocked
	if got := Classify(o); got != VerdictOSBlockingMeridian {
		t.Errorf("got %q", got)
	}
}

func TestClassify_ClientIsolationWhenProvenNotBlocked(t *testing.T) {
	o := healthy()
	o.PeersReached = 0
	o.OSBlock = OSBlockNotBlocked
	if got := Classify(o); got != VerdictClientIsolation {
		t.Errorf("got %q", got)
	}
}

func TestDescribe_OSBlockingCopyPerOS(t *testing.T) {
	steps := []StepResult{{Name: "gateway", Status: "pass", Detail: "ok"}}

	mac := Observations{GOOS: "darwin", Steps: steps}
	r := Describe(mac, VerdictOSBlockingMeridian)
	all := r.Statement + " " + strings.Join(r.Causes, " ") + " " + strings.Join(r.Fixes, " ")
	if !strings.Contains(all, "Local Network") {
		t.Errorf("macOS copy must name Local Network settings: %q", all)
	}
	if r.Verdict != VerdictOSBlockingMeridian || len(r.Steps) != 1 {
		t.Errorf("verdict/steps not attached: %+v", r)
	}

	win := Describe(Observations{GOOS: "windows"}, VerdictOSBlockingMeridian)
	if !strings.Contains(strings.ToLower(strings.Join(win.Fixes, " ")), "firewall") {
		t.Errorf("windows fix must mention firewall: %v", win.Fixes)
	}

	lin := Describe(Observations{GOOS: "linux"}, VerdictLocalAccessBlocked)
	causes := strings.ToLower(strings.Join(lin.Causes, " "))
	if len(lin.Causes) < 2 || !strings.Contains(causes, "isolat") || !strings.Contains(causes, "firewall") {
		t.Errorf("linux LOCAL_ACCESS_BLOCKED must list both causes: %v", lin.Causes)
	}
	if !strings.Contains(strings.Join(lin.Fixes, " "), "sudo") {
		t.Errorf("linux fixes should mention sudo re-run: %v", lin.Fixes)
	}
}

func TestDescribe_EveryVerdictHasCopy(t *testing.T) {
	vs := []Verdict{VerdictDHCPFailed, VerdictGatewayUnreachable, VerdictCaptivePortal, VerdictInternetDown,
		VerdictOSBlockingMeridian, VerdictClientIsolation, VerdictLocalAccessBlocked, VerdictHostDown,
		VerdictAllClear, VerdictInconclusive}
	for _, goos := range []string{"darwin", "windows", "linux"} {
		for _, v := range vs {
			r := Describe(Observations{GOOS: goos, ProbeError: "x"}, v)
			if r.Statement == "" || r.Verdict != v {
				t.Errorf("%s/%s: empty statement or wrong verdict", goos, v)
			}
			if v != VerdictAllClear && len(r.Fixes) == 0 {
				t.Errorf("%s/%s: no fixes", goos, v)
			}
		}
	}
}
