package meshcore

import (
	"fmt"
	"sort"
	"strconv"
)

// Finding is one mesh-health call-out, mirroring the Meshtastic tool's split:
// "problem" is something on THIS node Meridian can prove and you can fix;
// "check" is something it can't see from here (a peer's settings, range).
type Finding struct {
	Level string `json:"level"`
	Title string `json:"title"`
	Fix   string `json:"fix,omitempty"`
}

// diagnose turns a snapshot into mesh-health findings. MeshCore reports no
// airtime metric, so — unlike the Meshtastic tool — there's no "measured ~0%
// traffic" inference to make; isolation advice names the radio parameters a peer
// must match and stops there rather than guessing range vs. settings.
func diagnose(s Snapshot) []Finding {
	var fs []Finding
	r := s.Radio

	// A frequency of 0 almost always means it wasn't set, but it can also mean
	// the firmware didn't report it — so this stays a "check", not a verdict.
	if r == nil || r.FreqMHz == 0 {
		fs = append(fs, Finding{
			Level: "check",
			Title: "No radio frequency was read from this node.",
			Fix:   "If it isn't reaching any mesh, confirm its LoRa frequency, bandwidth and spreading factor are set on the device to your region's MeshCore values.",
		})
	}

	if len(s.Contacts) == 0 {
		fix := "A peer has to use the same LoRa radio settings, be powered on, and be within range. MeshCore nodes also only appear here once they've advertised and this node has received that advert."
		if r != nil && r.FreqMHz > 0 {
			fix = fmt.Sprintf("A peer has to match your frequency (%s MHz), bandwidth (%s kHz), spreading factor %d and coding rate %d, be powered on, and be within LoRa range. MeshCore nodes also only appear here once they've advertised and this node has received that advert.",
				trimFloat(r.FreqMHz), trimFloat(r.BandwidthKHz), r.SpreadFactor, r.CodingRate)
		}
		fs = append(fs, Finding{
			Level: "check",
			Title: "This node knows no other nodes yet.",
			Fix:   fix,
		})
	}

	return fs
}

// rankContacts floats the most recently heard contacts to the top.
func rankContacts(cs []Contact) {
	sort.SliceStable(cs, func(i, j int) bool {
		if cs[i].LastSeen != cs[j].LastSeen {
			return cs[i].LastSeen > cs[j].LastSeen
		}
		return cs[i].Name < cs[j].Name
	})
}

// trimFloat formats a float without trailing zeros: 910.525 → "910.525", 250 → "250".
func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
