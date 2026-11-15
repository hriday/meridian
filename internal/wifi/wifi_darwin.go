//go:build darwin

package wifi

import (
	"context"
	"fmt"
	"os/exec"
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
	// Refine the redaction note now that we know the authorization state.
	if info.SSIDRedacted {
		switch info.LocationStatus {
		case "authorized":
			info.Note = "SSIDs are still hidden even though Location Services is authorized — on an unsigned build macOS can withhold them. Re-scan, or try a signed build."
		case "denied", "restricted":
			info.Note = "SSIDs are hidden because Location Services is turned off for Meridian. Enable it in System Settings › Privacy & Security › Location Services. The RF metrics are unaffected."
		default: // notDetermined / unknown
			info.Note = "macOS hides Wi-Fi network names until you grant Meridian Location Services access. The RF metrics (channel, signal, SNR) are unaffected."
		}
	}
	return info, nil
}
