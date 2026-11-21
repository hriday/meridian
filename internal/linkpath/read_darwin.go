//go:build darwin

package linkpath

import (
	"os/exec"
)

// readLink reads link speed/duplex (from ifconfig) and whether the interface is
// Wi-Fi (from networksetup's hardware-port map). Read-only.
func readLink(iface string) linkState {
	var ls linkState
	if iface == "" {
		return ls
	}
	if out, err := exec.Command("ifconfig", iface).Output(); err == nil {
		ls = parseIfconfigMedia(string(out))
	}
	// Reliable Wi-Fi detection: ifconfig alone can't tell a down Ethernet from
	// Wi-Fi, but the hardware-port map names the port type.
	if out, err := exec.Command("networksetup", "-listallhardwareports").Output(); err == nil {
		if port, ok := parseHardwarePorts(string(out))[iface]; ok && isWirelessPort(port) {
			ls.wireless = true
		}
	}
	return ls
}
