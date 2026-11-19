// Package netdiag implements the Local Network Diagnostic recipe: a
// deterministic decision tree that turns a handful of observed facts into one
// plain-language verdict. This file is pure logic (no OS, network or time
// access); probes live in sibling files.
package netdiag

// Verdict names the single conclusion of a diagnostic run.
type Verdict string

const (
	VerdictDHCPFailed         Verdict = "dhcp_failed"
	VerdictGatewayUnreachable Verdict = "gateway_unreachable"
	VerdictCaptivePortal      Verdict = "captive_portal"
	VerdictInternetDown       Verdict = "internet_down"
	VerdictOSBlockingMeridian Verdict = "os_blocking_meridian"
	VerdictClientIsolation    Verdict = "client_isolation"
	VerdictLocalAccessBlocked Verdict = "local_access_blocked"
	VerdictHostDown           Verdict = "host_down"
	VerdictAllClear           Verdict = "all_clear"
	VerdictInconclusive       Verdict = "inconclusive"
)

// OSBlockState is the tri-state result of the "is this OS blocking Meridian
// specifically" check. The zero value means the check was not run.
type OSBlockState int

const (
	OSBlockUnknown OSBlockState = iota
	OSBlockBlocked
	OSBlockNotBlocked
	OSBlockCouldNotDetermine
)

// StepResult is one probe step. Status is pass|fail|skip|error.
type StepResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// Observations are the facts gathered by the probes.
type Observations struct {
	GOOS                                   string
	IdentityOK                             bool // true only when the addressing probe succeeded
	HasGateway, IsLinkLocal, IPv6Only      bool
	GatewayReachable                       bool
	InternetReachable, InternetIntercepted bool
	PeerSampleConfident                    bool
	PeersTried, PeersReached               int
	TargetGiven, TargetReached             bool
	OSBlock                                OSBlockState
	ProbeError                             string // non-empty means a required probe errored
	Steps                                  []StepResult
}

// Report is the human-facing result.
type Report struct {
	Verdict   Verdict      `json:"verdict"`
	Statement string       `json:"statement"`
	Causes    []string     `json:"causes"`
	Fixes     []string     `json:"fixes"`
	Steps     []StepResult `json:"steps"`
}

// Classify applies the verdict table. It is pure and deterministic.
func Classify(o Observations) Verdict {
	// Without a successful addressing probe nothing downstream is trustworthy.
	if !o.IdentityOK {
		return VerdictInconclusive
	}
	// IPv6-only hosts have no routable IPv4 gateway by design; this must come
	// before the DHCP row so they are never mislabeled DHCP_FAILED.
	if o.IPv6Only {
		return VerdictInconclusive
	}
	if !o.HasGateway || o.IsLinkLocal {
		return VerdictDHCPFailed
	}
	if !o.GatewayReachable {
		return VerdictGatewayUnreachable
	}
	// Past addressing: an errored internet/peer probe never yields a confident
	// guess (e.g. ICMP unavailable must not read as host down).
	if o.ProbeError != "" {
		return VerdictInconclusive
	}

	peers := o.PeersReached
	if o.TargetGiven && o.TargetReached && peers == 0 {
		peers = 1 // a reached target is a reached peer
	}

	if !o.InternetReachable {
		if o.InternetIntercepted {
			return VerdictCaptivePortal
		}
		if peers > 0 {
			return VerdictInternetDown
		}
		return VerdictInconclusive
	}

	if peers == 0 {
		// Zero peers is only trusted when the sample was confident.
		if !o.PeerSampleConfident {
			return VerdictInconclusive
		}
		switch o.OSBlock {
		case OSBlockBlocked:
			return VerdictOSBlockingMeridian
		case OSBlockNotBlocked:
			return VerdictClientIsolation
		default: // CouldNotDetermine, or the check never ran
			return VerdictLocalAccessBlocked
		}
	}

	if o.TargetGiven && !o.TargetReached {
		return VerdictHostDown
	}
	return VerdictAllClear
}

// Describe fills the statement, causes and fixes for v and attaches o.Steps.
func Describe(o Observations, v Verdict) Report {
	r := Report{Verdict: v, Steps: o.Steps, Causes: []string{}, Fixes: []string{}}
	switch v {
	case VerdictDHCPFailed:
		r.Statement = "This machine is not actually on a network: it has no gateway or only a self-assigned (link-local) address."
		r.Causes = []string{"The DHCP server did not hand out an address.", "Not connected to Wi-Fi or Ethernet, or the link is down."}
		r.Fixes = []string{"Reconnect to the network or toggle the adapter off and on.", "Check that the router's DHCP server is running."}
	case VerdictGatewayUnreachable:
		r.Statement = "Connected to a network, but the router (gateway) is not answering."
		r.Causes = []string{"Weak or unstable Wi-Fi/cable link.", "The router is hung or rebooting."}
		r.Fixes = []string{"Move closer to the access point or check the cable.", "Restart the router."}
	case VerdictCaptivePortal:
		r.Statement = "A login page appears to be intercepting your traffic (captive portal)."
		r.Causes = []string{"Hotel, airport or guest Wi-Fi requires sign-in before granting internet access."}
		r.Fixes = []string{"Open a browser to any http:// site and complete the sign-in or terms page."}
	case VerdictInternetDown:
		r.Statement = "Your local network is fine, but the internet is unreachable."
		r.Causes = []string{"The ISP or WAN link is down.", "The router lost its upstream connection."}
		r.Fixes = []string{"Restart the modem/router.", "Check the ISP status page or contact the provider."}
	case VerdictOSBlockingMeridian:
		r.Statement = "This machine is blocking Meridian from reaching local devices, while the gateway and internet are fine."
		switch o.GOOS {
		case "darwin":
			r.Causes = []string{"macOS Local Network privacy has not been granted to Meridian."}
			r.Fixes = []string{"Open System Settings > Privacy & Security > Local Network and enable Meridian, then re-run."}
		case "windows":
			r.Causes = []string{"Windows Defender Firewall is blocking Meridian, or the network profile is Public."}
			r.Fixes = []string{"Allow Meridian through Windows Defender Firewall.", "Set the network profile to Private."}
		default:
			r.Causes = []string{"A host firewall on this machine is blocking Meridian's local traffic."}
			r.Fixes = []string{"Review ufw/firewalld/iptables rules for outbound local traffic and allow it."}
		}
	case VerdictClientIsolation:
		r.Statement = "No other device on this network is reachable, even with a baseline check outside Meridian: the network isolates clients."
		r.Causes = []string{"Hotspot or guest Wi-Fi with client (AP) isolation enabled."}
		r.Fixes = []string{"Switch to a non-guest network or disable client isolation on the access point/router."}
	case VerdictLocalAccessBlocked:
		r.Statement = "Meridian can't reach any device on this local network (the gateway and internet are fine)."
		r.Causes = []string{
			"The network is isolating clients (guest Wi-Fi, hotspot or AP isolation).",
			"Something on this machine is blocking Meridian's local access (OS privacy setting or host firewall).",
		}
		switch o.GOOS {
		case "darwin":
			r.Fixes = []string{"Open System Settings > Privacy & Security > Local Network and enable Meridian."}
		case "windows":
			r.Fixes = []string{"Allow Meridian through Windows Defender Firewall and set the network to Private."}
		default:
			r.Fixes = []string{"Check whether ufw/firewalld/iptables is dropping local traffic."}
		}
		r.Fixes = append(r.Fixes,
			"Re-run with sudo (elevated) so Meridian can tell which cause applies.",
			"If on guest Wi-Fi or a hotspot, join the main network or disable client isolation.")
	case VerdictHostDown:
		r.Statement = "The network is fine, but the host you asked about is not answering."
		r.Causes = []string{"The host is powered off, asleep or unplugged.", "The host's firewall drops probes, or the address is wrong."}
		r.Fixes = []string{"Check the host is on and on the same network.", "Verify the address and any firewall on the host."}
	case VerdictAllClear:
		r.Statement = "Everything checks out: gateway, internet and local devices are all reachable."
	default:
		r.Verdict = VerdictInconclusive
		r.Statement = "Not enough reliable evidence to name a cause."
		r.Causes = []string{"A probe failed or returned too little data to trust."}
		if o.ProbeError != "" {
			r.Causes = append(r.Causes, o.ProbeError)
		}
		if o.IPv6Only {
			r.Causes = append(r.Causes, "This network appears to be IPv6-only, which this check can't fully assess.")
		}
		r.Fixes = []string{"Re-run the diagnostic, optionally elevated, or with a specific target host."}
	}
	return r
}
