//go:build windows

package wifi

import (
	"context"
	"fmt"
	"os/exec"
)

// Supported reports whether Wi-Fi analysis works on this OS.
func Supported() bool { return true }

// Status reads the Wi-Fi environment from `netsh wlan`: the connected network
// from `show interfaces`, and neighbors from `show networks mode=bssid`. No cgo,
// no admin. netsh reports signal as a 0–100 quality percentage (not dBm) and no
// noise, so those are approximate/absent (parseNetsh sets a Note). Field labels
// are English-locale; the native WLAN API is a later upgrade.
func Status(ctx context.Context) (Info, error) {
	iface, _ := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces").Output()
	nets, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "networks", "mode=bssid").Output()
	if err != nil {
		return Info{}, fmt.Errorf("couldn't read Wi-Fi via netsh (is the WLAN AutoConfig service running and Wi-Fi on?): %w", err)
	}
	info := parseNetsh(string(iface), string(nets))
	if info.Current.SSID == "" && len(info.Neighbors) == 0 {
		return info, fmt.Errorf("no Wi-Fi networks found")
	}
	return info, nil
}
