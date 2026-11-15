//go:build darwin

package wifi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Supported reports whether Wi-Fi analysis works on this OS.
func Supported() bool { return true }

// Status reads the Wi-Fi environment from `system_profiler SPAirPortDataType`.
// No cgo, no privileges. SSIDs are redacted by macOS without Location Services;
// the RF metrics are unaffected (Info.Note flags this).
func Status(ctx context.Context) (Info, error) {
	out, err := exec.CommandContext(ctx, "system_profiler", "SPAirPortDataType").Output()
	if err != nil {
		return Info{}, fmt.Errorf("couldn't read Wi-Fi info (is Wi-Fi on?): %w", err)
	}
	info := parseAirport(string(out))
	if info.Interface == "" {
		return info, fmt.Errorf("no Wi-Fi interface found")
	}
	info.LocationStatus = locationStatus()

	// system_profiler runs in a separate process and may redact the SSID even
	// when Meridian holds the Location grant. CoreWLAN reads the connected
	// network's name in our own process, where our grant applies, so fill the
	// current SSID from it when system_profiler gave us "<redacted>".
	if isRedacted(info.Current.SSID) {
		if ssid := currentSSID(); ssid != "" && !isRedacted(ssid) {
			info.Current.SSID = ssid
		}
	}

	// Recompute redaction across the connected network and its neighbors, then
	// tailor the note to what's actually hidden and the authorization state.
	currentHidden := isRedacted(info.Current.SSID)
	neighborsHidden := false
	for _, n := range info.Neighbors {
		if isRedacted(n.SSID) {
			neighborsHidden = true
			break
		}
	}
	info.SSIDRedacted = currentHidden || neighborsHidden
	info.Note = wifiRedactionNote(info.LocationStatus, currentHidden, neighborsHidden)
	return info, nil
}

func isRedacted(ssid string) bool { return strings.Contains(ssid, "redacted") }

// wifiRedactionNote explains what's hidden and why, given the authorization
// state and which names (connected / neighbors) are still redacted.
func wifiRedactionNote(status string, currentHidden, neighborsHidden bool) string {
	if !currentHidden && !neighborsHidden {
		return ""
	}
	switch status {
	case "authorized":
		if currentHidden {
			return "Network names are still hidden even though Location Services is authorized — on an unsigned build macOS can withhold them. Re-scan, or try a signed build."
		}
		return "Your connected network name is shown. Neighboring names stay hidden until a full Wi-Fi scan is permitted (a signed build helps). The RF metrics are unaffected."
	case "denied", "restricted":
		return "Network names are hidden because Location Services is turned off for Meridian. Enable it in System Settings › Privacy & Security › Location Services. The RF metrics are unaffected."
	default: // notDetermined / unknown
		return "macOS hides Wi-Fi network names until you grant Meridian Location Services access. The RF metrics (channel, signal, SNR) are unaffected."
	}
}
