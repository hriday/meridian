//go:build linux

package wifi

import (
	"context"
	"fmt"
	"os/exec"
)

// Supported reports whether Wi-Fi analysis works on this OS.
func Supported() bool { return true }

// Status reads the Wi-Fi environment from NetworkManager's `nmcli`. No cgo, no
// root. nmcli reports signal as a 0–100 quality percentage (not dBm) and gives
// no noise or channel width, so those are approximate/absent (parseNmcli sets a
// Note). A missing nmcli (no NetworkManager) surfaces as an error the caller
// shows.
func Status(ctx context.Context) (Info, error) {
	out, err := exec.CommandContext(ctx, "nmcli", "-t", "-f",
		"ACTIVE,SSID,BSSID,CHAN,FREQ,SIGNAL,SECURITY", "dev", "wifi").Output()
	if err != nil {
		return Info{}, fmt.Errorf("couldn't read Wi-Fi via nmcli (is NetworkManager installed and Wi-Fi on?): %w", err)
	}
	info := parseNmcli(string(out))
	if info.Current.SSID == "" && len(info.Neighbors) == 0 {
		return info, fmt.Errorf("no Wi-Fi networks found")
	}
	return info, nil
}
