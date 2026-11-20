package netdiag

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/myip"
	"github.com/hriday/meridian/internal/netif"
	"github.com/hriday/meridian/internal/netroute"
	"github.com/hriday/meridian/internal/scan"
)

// Options configures a diagnostic run.
type Options struct {
	// Target is an optional host (IP or name) the user specifically cares about.
	Target string
}

const (
	// minLikelyPeers (K) is how many distinct, independently-likely-up peers
	// (neighbor-table entries) must have been probed and failed before "no
	// peer answered" is trusted. Below it the verdict is INCONCLUSIVE.
	minLikelyPeers = 2
	// maxSample bounds how many peers are probed, to stay gentle.
	maxSample = 8
	// subnetFill is how many guessed in-subnet hosts are added as a last
	// resort when the target and neighbor table give fewer candidates.
	subnetFill = 4
	// maxOSBlockPeers bounds peers handed to the OS-block check.
	maxOSBlockPeers = 4
)

// Probe seams: package-level so tests fake them without touching the host.
var (
	probeIdentity  = realIdentity
	probeReach     = realReach
	sampleInternet = realInternet
	neighborsFn    = Neighbors
	osBlockFn      = OSBlockCheck
	resolveHost    = realResolve
)

// Run executes the diagnostic steps in order, streaming each via onStep
// (nil-safe), and returns the classified report.
func Run(ctx context.Context, opts Options, onStep func(StepResult)) (Report, error) {
	obs := Observations{GOOS: runtime.GOOS}
	emit := func(name, status, detail string) {
		s := StepResult{Name: name, Status: status, Detail: detail}
		obs.Steps = append(obs.Steps, s)
		if onStep != nil {
			onStep(s)
		}
	}
	finish := func() (Report, error) { return Describe(obs, Classify(obs)), nil }

	// 1. identity
	ip, gw, iface, prefix, linkLocal, ipv6Only, hasGW, err := probeIdentity()
	if err != nil {
		obs.ProbeError = "could not read this machine's network identity: " + err.Error()
		emit("identity", "error", obs.ProbeError)
		return finish()
	}
	obs.IdentityOK = true
	obs.HasGateway, obs.IsLinkLocal, obs.IPv6Only = hasGW, linkLocal, ipv6Only
	emit("identity", "pass", fmt.Sprintf("%s on %s, gateway %s", ip, iface, gw))
	if ipv6Only || !hasGW || linkLocal {
		return finish() // downstream steps are meaningless without IPv4 addressing
	}

	// 2. gateway
	reached, avail := probeReach(ctx, gw)
	obs.GatewayReachable = reached
	switch {
	case reached:
		emit("gateway", "pass", gw+" answered")
	case !avail:
		obs.ProbeError = "could not test the gateway (no usable probe)"
		emit("gateway", "error", obs.ProbeError)
	default:
		emit("gateway", "fail", gw+" did not answer")
	}
	if !reached {
		return finish()
	}

	// 3. internet
	obs.InternetReachable, obs.InternetIntercepted = sampleInternet(ctx)
	switch {
	case obs.InternetReachable:
		emit("internet", "pass", "internet reachable")
	case obs.InternetIntercepted:
		emit("internet", "fail", "traffic is being intercepted (captive portal)")
	default:
		emit("internet", "fail", "internet not reachable")
	}

	// 4. peers: target -> neighbor table -> bounded subnet guess.
	var seeded []string // independently-likely-up: neighbors
	var sample []string
	seen := map[string]bool{gw: true, ip: true} // gateway and self never count as peers
	add := func(s string, likely bool) bool {
		if seen[s] || len(sample) >= maxSample {
			return false
		}
		seen[s] = true
		sample = append(sample, s)
		if likely {
			seeded = append(seeded, s)
		}
		return true
	}

	target := ""
	if opts.Target != "" {
		t, terr := validateTarget(ctx, opts.Target)
		if terr != nil {
			// A failed lookup is an error, never a verdict about the host.
			obs.ProbeError = "could not resolve target: " + terr.Error()
		} else {
			target = t
			obs.TargetGiven = true
			if t != gw && t != ip {
				add(t, false)
			}
		}
	}
	if ns, nerr := neighborsFn(ctx); nerr == nil {
		// reachable entries first, then stale; State is only a seed.
		for _, want := range []string{"reachable", "stale"} {
			for _, n := range ns {
				if a, perr := netip.ParseAddr(n.IP); perr == nil && a.Is4() && n.State == want {
					add(a.String(), true)
				}
			}
		}
	}
	if len(sample) < subnetFill {
		for _, h := range subnetHosts(ip, prefix, subnetFill+2) {
			if len(sample) >= subnetFill {
				break
			}
			add(h, false)
		}
	}

	type res struct{ reached, avail bool }
	results := make([]res, len(sample))
	var wg sync.WaitGroup
	for i, p := range sample {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			r, a := probeReach(ctx, p)
			results[i] = res{r, a}
		}(i, p)
	}
	wg.Wait()

	reachedAny, failedLikely, triedFailed := 0, 0, 0
	likely := map[string]bool{}
	for _, s := range seeded {
		likely[s] = true
	}
	for i, p := range sample {
		r := results[i]
		if !r.avail {
			continue // could not test: neither up nor down
		}
		if p == target {
			obs.TargetReached = r.reached
			if !r.reached {
				triedFailed++
				failedLikely++ // the user's own failed target is a likely-up peer
			}
			continue
		}
		obs.PeersTried++
		if r.reached {
			obs.PeersReached++
			reachedAny++
		} else {
			triedFailed++
			if likely[p] {
				failedLikely++
			}
		}
	}
	if target != "" && (target == gw || target == ip) {
		obs.TargetReached = true // the gateway already answered; we are ourselves
	}
	obs.PeerSampleConfident = obs.PeersReached == 0 && failedLikely >= minLikelyPeers
	if obs.PeersReached > 0 || obs.TargetReached {
		emit("peers", "pass", fmt.Sprintf("%d of %d peers answered", obs.PeersReached, obs.PeersTried))
	} else {
		emit("peers", "fail", fmt.Sprintf("0 of %d peers answered", obs.PeersTried))
	}

	// 5. OS-block check: whenever at least one peer Meridian itself probed
	// failed (the explicit target counts). A positive result is definitive,
	// so this is not gated on sample confidence.
	if obs.InternetReachable && obs.PeersReached == 0 && !obs.TargetReached && triedFailed > 0 {
		peers := sample
		if len(peers) > maxOSBlockPeers {
			peers = peers[:maxOSBlockPeers]
		}
		st, detail := osBlockFn(ctx, peers) // peers are all validated IPs
		obs.OSBlock = st
		status := "pass"
		if st == OSBlockBlocked {
			status = "fail"
		} else if st == OSBlockCouldNotDetermine {
			status = "skip"
		}
		emit("osblock", status, detail)
	}
	return finish()
}

// validateTarget returns a literal IP for t, resolving hostnames. Raw user
// text is never returned, since peers flow into OS command lines.
func validateTarget(ctx context.Context, t string) (string, error) {
	t = strings.TrimSpace(t)
	if a, err := netip.ParseAddr(t); err == nil {
		return cleanIPv4(a)
	}
	ip, err := resolveHost(ctx, t)
	if err != nil {
		return "", err
	}
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return "", err
	}
	return cleanIPv4(a)
}

// cleanIPv4 accepts only a plain, zone-free IPv4 (v1 is IPv4-only). ParseAddr
// accepts "fe80::1%anything", so the zone text must never reach a shell.
func cleanIPv4(a netip.Addr) (string, error) {
	if !a.Is4() || a.Zone() != "" {
		return "", fmt.Errorf("%q is not a plain IPv4 address", a.String())
	}
	return a.String(), nil
}

func realResolve(ctx context.Context, host string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupNetIP(cctx, "ip4", host)
	if err != nil || len(ips) == 0 {
		if err == nil {
			err = fmt.Errorf("no IPv4 address for %q", host)
		}
		return "", err
	}
	return ips[0].String(), nil
}

// subnetHosts returns up to n low usable host addresses of ip/prefix.
func subnetHosts(ip, prefix string, n int) []string {
	bits, err := strconv.Atoi(prefix)
	a, perr := netip.ParseAddr(ip)
	if err != nil || perr != nil || !a.Is4() || bits < 16 || bits > 30 {
		return nil
	}
	p, _ := a.Prefix(bits)
	var out []string
	for c := p.Addr().Next(); p.Contains(c) && len(out) < n; c = c.Next() {
		out = append(out, c.String())
	}
	return out
}

func realIdentity() (ip, gw, iface, prefix string, linkLocal, ipv6Only, hasGW bool, err error) {
	info, err := myip.Local()
	if err != nil {
		return
	}
	ip, iface, gw = info.PrimaryIP, info.PrimaryIf, info.Gateway
	if gw == "" {
		gw = netroute.DefaultGateway()
	}
	hasGW = gw != ""
	ifaces, lerr := netif.List()
	if lerr != nil {
		err = lerr
		return
	}
	if a, perr := netip.ParseAddr(ip); perr == nil {
		linkLocal = a.IsLinkLocalUnicast()
	}
	for _, f := range ifaces {
		if f.Name != iface && ip != "" {
			continue
		}
		for _, c := range f.IPv4 {
			if pf, perr := netip.ParsePrefix(c); perr == nil && (ip == "" || pf.Addr().String() == ip) {
				prefix = strconv.Itoa(pf.Bits())
			}
		}
	}
	if ip == "" {
		for _, f := range ifaces {
			if !f.Up || f.Loopback {
				continue
			}
			for _, c := range f.IPv6 {
				if pf, perr := netip.ParsePrefix(c); perr == nil && !pf.Addr().IsLinkLocalUnicast() && !pf.Addr().IsLoopback() {
					ipv6Only = true
				}
			}
		}
	}
	return
}

var (
	pingerOnce sync.Once
	pinger     *scan.ICMPPinger
	// fallbackPorts: a refused connect (closed) still proves the host is up.
	fallbackPorts = []int{443, 80, 22, 445, 53}
)

// realReach tries unprivileged ICMP, then falls back to TCP connect when ICMP
// is unavailable or unanswered. available=false from ICMP only triggers the
// fallback; it is never recorded as "down".
func realReach(ctx context.Context, host string) (reached, available bool) {
	a, err := netip.ParseAddr(host)
	if err != nil {
		return false, false
	}
	pingerOnce.Do(func() { pinger = scan.NewICMPPinger() })
	if _, ok, avail := pinger.Ping(ctx, a, time.Second); avail && ok {
		return true, true
	}
	rs, _ := scan.ScanPorts(ctx, host, fallbackPorts, time.Second, len(fallbackPorts), nil)
	for _, r := range rs {
		if r.State == "open" || r.State == "closed" {
			return true, true
		}
	}
	return false, true // TCP ran and nothing answered
}

// realInternet checks fixed anycast TCP endpoints; if they fail it looks for
// an HTTP response that is not the expected 204 (a captive portal).
func realInternet(ctx context.Context) (reached, intercepted bool) {
	for _, h := range []string{"1.1.1.1", "8.8.8.8"} {
		rs, _ := scan.ScanPorts(ctx, h, []int{443}, 2*time.Second, 1, nil)
		for _, r := range rs {
			if r.State == "open" {
				return true, false
			}
		}
	}
	c := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://connectivitycheck.gstatic.com/generate_204", nil)
	if err != nil {
		return false, false
	}
	resp, err := c.Do(req)
	if err != nil {
		return false, false
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return true, false // HTTP path is clean even though anycast 443 failed
	}
	return false, true
}
