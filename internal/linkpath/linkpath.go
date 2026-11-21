// Package linkpath answers one question a host normally can't: is there a hub
// or switch between this machine and the router? It can't be answered with
// certainty — an unmanaged switch has no IP, speaks no protocol, and doesn't
// change anything a host can see — so linkpath gathers the signals that DO
// betray an intermediate L2 device and reports an honest, hedged verdict:
//
//   - Link duplex. A real hub shares the medium and forces half-duplex; modern
//     switches and routers are full-duplex. A half-duplex link is a strong hub
//     tell. This read is host-local and needs no privilege.
//   - LLDP / CDP. A managed switch announces itself — name, model, the exact
//     port you're on. Capturing those frames needs elevated privilege; without
//     it linkpath says so rather than implying none is there.
//
// What it cannot do: prove a switch is absent, count switches in the path, or
// see anything on a Wi-Fi link (there's an access point, and usually a switch,
// upstream, but a host can't see past the AP). The verdict always names its own
// blind spots.
package linkpath

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/lldp"
	"github.com/hriday/meridian/internal/myip"
)

// DefaultLLDPWindow is how long to listen for a switch's LLDP/CDP advertisement.
// LLDP transmits about every 30s, so a shorter window risks missing a switch
// that is in fact present; this matches the standalone lldp tool.
const DefaultLLDPWindow = 35 * time.Second

// Report is one point-in-time inspection of the link toward the router.
type Report struct {
	Iface      string          `json:"iface"`             // egress interface toward the gateway
	Gateway    string          `json:"gateway,omitempty"` // default gateway, if known
	Wireless   bool            `json:"wireless"`          // the egress link is Wi-Fi
	LinkKnown  bool            `json:"linkKnown"`         // speed/duplex were readable
	SpeedMbps  int             `json:"speedMbps"`         // 0 if unknown
	Duplex     string          `json:"duplex"`            // "full" | "half" | "" (unknown)
	Media      string          `json:"media,omitempty"`   // raw media string, for reference
	LLDPStatus string          `json:"lldpStatus"`        // ok | none-heard | needs-privilege | unsupported | skipped-wireless | error
	Neighbors  []lldp.Neighbor `json:"neighbors"`         // switches/routers that announced themselves
	Verdict    string          `json:"verdict"`           // the one-line synthesis
	Findings   []Finding       `json:"findings"`          // the reasons behind the verdict
	TakenAt    string          `json:"takenAt"`           // RFC3339 UTC
}

// Finding is one signal or caveat. Level is "found" (a device/condition
// detected), "good" (a clean signal), "info", or "limit" (a blind spot).
type Finding struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// linkState is the host-local link read, the input to the pure assessor.
type linkState struct {
	speedMbps int
	duplex    string // "full" | "half" | ""
	wireless  bool
	media     string
	known     bool
}

// Inspect reads the egress link and listens for a switch's LLDP/CDP, then
// synthesizes a verdict. Read-only. The duplex read needs no privilege; the
// LLDP capture does, and its absence is reported, never hidden.
func Inspect(ctx context.Context, lldpWindow time.Duration) (Report, error) {
	info, err := myip.Local()
	if err != nil {
		return Report{}, err
	}
	iface := info.PrimaryIf
	ls := readLink(iface)

	var neigh []lldp.Neighbor
	status := "skipped"
	switch {
	case ls.wireless:
		status = "skipped-wireless"
	case !lldp.Supported():
		status = "unsupported"
	default:
		if lldpWindow <= 0 {
			lldpWindow = DefaultLLDPWindow
		}
		n, lerr := lldp.Listen(ctx, iface, lldpWindow)
		switch {
		case errors.Is(lerr, lldp.ErrNeedsPrivilege):
			status = "needs-privilege"
		case lerr != nil:
			status = "error"
		case len(n) == 0:
			status = "none-heard"
		default:
			neigh = n
			status = "ok"
		}
	}

	verdict, findings := assess(ls, neigh, status, info.Gateway)
	return Report{
		Iface:      iface,
		Gateway:    info.Gateway,
		Wireless:   ls.wireless,
		LinkKnown:  ls.known,
		SpeedMbps:  ls.speedMbps,
		Duplex:     ls.duplex,
		Media:      ls.media,
		LLDPStatus: status,
		Neighbors:  neigh,
		Verdict:    verdict,
		Findings:   findings,
		TakenAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// assess is the pure synthesis: given the link read, the LLDP neighbours and the
// capture status, it produces the headline verdict and the findings behind it.
// Testable without any hardware.
func assess(ls linkState, neigh []lldp.Neighbor, lldpStatus, gateway string) (string, []Finding) {
	var fs []Finding

	// Wireless: the hub/switch question doesn't apply to the radio hop. (LLDP is
	// never captured on a wireless egress, so there are no neighbours to list.)
	if ls.wireless {
		fs = append(fs, Finding{"info", "This machine reaches the network over Wi-Fi. “Hub vs switch” is a wired-link question — it doesn't apply to the wireless hop."})
		fs = append(fs, Finding{"limit", "There's an access point (and almost always a switch) upstream of the AP, but a host can't see past the access point to tell."})
		return "You're on Wi-Fi — a host can't see whether a hub or switch sits upstream of the access point.", fs
	}

	// A half-duplex reading at gigabit or faster is physically impossible
	// (gigabit mandates full-duplex) — it's a driver/virtual-adapter misreport,
	// not a real hub, so don't let it drive a false hub verdict.
	if ls.duplex == "half" && ls.speedMbps >= 1000 {
		ls.duplex = ""
	}

	// Wired link read (duplex is the hub tell).
	switchSeen := false
	hubSeen := false

	if ls.known {
		speed := ""
		if ls.speedMbps > 0 {
			speed = formatSpeed(ls.speedMbps) + ", "
		}
		switch ls.duplex {
		case "half":
			hubSeen = true
			fs = append(fs, Finding{"found", "Half-duplex link (" + speed + "half duplex). A hub shares the medium and forces half-duplex, so there's almost certainly a hub between you and the router."})
		case "full":
			fs = append(fs, Finding{"good", "Full-duplex link (" + speed + "full duplex). No hub detected this way — a hub would force half-duplex."})
		default:
			fs = append(fs, Finding{"info", "Link is up (" + strings.TrimSuffix(speed, ", ") + ") but the driver didn't report duplex, so the hub-by-duplex check is inconclusive."})
		}
	} else {
		fs = append(fs, Finding{"limit", "Couldn't read this link's speed/duplex on this system, so the hub-by-duplex check is unavailable."})
	}

	// LLDP / CDP: a managed switch (or the router) that announced itself.
	switch lldpStatus {
	case "ok":
		for _, n := range neigh {
			f := neighborFinding(n, gateway)
			fs = append(fs, f)
			if f.Level == "found" && isSwitchy(n) && !isGateway(n, gateway) {
				switchSeen = true
			}
			if isRepeater(n) {
				hubSeen = true
			}
		}
	case "none-heard":
		fs = append(fs, Finding{"info", "No device announced itself via LLDP/CDP during the listen window. A managed switch may have LLDP turned off, and an unmanaged switch or hub never announces itself."})
	case "needs-privilege":
		fs = append(fs, Finding{"limit", "Couldn't listen for switch advertisements — raw capture needs elevated privilege. Re-run with privilege, or use the triage-l2 recipe, to detect a managed switch by LLDP/CDP."})
	case "unsupported":
		fs = append(fs, Finding{"limit", "LLDP/CDP capture isn't supported on this operating system yet, so a managed switch can't be detected here."})
	case "error":
		fs = append(fs, Finding{"limit", "The LLDP/CDP capture couldn't run, so a managed switch couldn't be detected this time."})
	}

	// The unmanaged-switch blind spot is always true — say so every time.
	fs = append(fs, Finding{"limit", "An unmanaged switch is invisible to every check here — no IP, no LLDP, no effect on duplex. Nothing below proves the path is switch-free."})

	return verdictLine(switchSeen, hubSeen, ls, lldpStatus), fs
}

// verdictLine turns the collected signals into the one-line headline.
func verdictLine(switchSeen, hubSeen bool, ls linkState, lldpStatus string) string {
	switch {
	case switchSeen:
		return "A switch is directly attached between you and the router (it announced itself via LLDP/CDP)."
	case hubSeen:
		return "A hub is almost certainly in the path (the link is half-duplex)."
	case ls.known && ls.duplex == "full" && (lldpStatus == "ok" || lldpStatus == "none-heard"):
		// LLDP capture actually ran (heard a non-switch, or nothing) — don't tell
		// the user to re-run with privilege they already supplied.
		return "No hub or announced switch detected — but an unmanaged switch could still be in the path, invisibly."
	case ls.known && ls.duplex == "full":
		return "No hub detected (full-duplex). Switch detection needs privilege — run the triage-l2 recipe for the full check."
	default:
		return "Inconclusive — not enough signal to say whether a hub or switch is in the path."
	}
}

// neighborFinding describes one LLDP/CDP neighbour, distinguishing the router
// itself from an intermediate switch.
func neighborFinding(n lldp.Neighbor, gateway string) Finding {
	name := n.SystemName
	if name == "" {
		name = n.ChassisID
	}
	where := ""
	if n.PortID != "" {
		where = " — you're on its port " + n.PortID
	}
	switch {
	case isRepeater(n):
		return Finding{"found", "A hub (repeater) announced itself via " + n.Protocol + ": " + name + where + "."}
	case isGateway(n, gateway):
		return Finding{"info", "The directly-attached device is your router itself (" + name + ", " + n.MgmtIP + ") — it announced no separate switch."}
	case isSwitchy(n):
		return Finding{"found", "A managed switch is directly attached (" + n.Protocol + "): " + name + where + "."}
	default:
		return Finding{"found", "A device is directly attached (" + n.Protocol + "): " + name + where + " — it didn't advertise a switch capability."}
	}
}

func isSwitchy(n lldp.Neighbor) bool {
	return hasCap(n, "bridge", "switch")
}

func isRepeater(n lldp.Neighbor) bool {
	return hasCap(n, "repeater")
}

// isGateway reports whether a neighbour IS the default gateway: a device whose
// management IP equals the gateway is the router itself (even a combined
// router+switch that also advertises a Bridge capability), not an intermediate
// switch between the host and the router.
func isGateway(n lldp.Neighbor, gateway string) bool {
	return gateway != "" && n.MgmtIP == gateway
}

func hasCap(n lldp.Neighbor, subs ...string) bool {
	for _, c := range n.Capabilities {
		lc := strings.ToLower(c)
		for _, s := range subs {
			if strings.Contains(lc, s) {
				return true
			}
		}
	}
	return false
}

func formatSpeed(mbps int) string {
	if mbps >= 1000 && mbps%1000 == 0 {
		return strconv.Itoa(mbps/1000) + " Gb/s"
	}
	return strconv.Itoa(mbps) + " Mb/s"
}
