package linkpath

import (
	"regexp"
	"strconv"
	"strings"
)

var mediaSpeed = regexp.MustCompile(`(\d+)base`)

// parseIfconfigMedia reads the "media:" line of `ifconfig <iface>` output and
// extracts speed and duplex. A wired Ethernet link reports a base-T media type
// and a duplex; Wi-Fi and down links don't, leaving the read "unknown" (the
// reader marks wireless separately). Build-tag-free so it is unit-tested on
// every platform.
func parseIfconfigMedia(out string) linkState {
	var ls linkState
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "media:") {
			continue
		}
		ls.media = strings.TrimSpace(strings.TrimPrefix(line, "media:"))
		low := strings.ToLower(line)
		switch {
		case strings.Contains(low, "half-duplex"):
			ls.duplex = "half"
		case strings.Contains(low, "full-duplex"):
			ls.duplex = "full"
		}
		if m := mediaSpeed.FindStringSubmatch(low); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				ls.speedMbps = n
			}
		}
	}
	ls.known = ls.duplex != "" || ls.speedMbps > 0
	return ls
}

// parseHardwarePorts maps interface device names to their hardware-port names
// from `networksetup -listallhardwareports`, so the reader can tell a Wi-Fi
// interface apart from wired Ethernet reliably.
func parseHardwarePorts(out string) map[string]string {
	m := map[string]string{}
	var port string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Hardware Port:"):
			port = strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))
		case strings.HasPrefix(line, "Device:"):
			dev := strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			if dev != "" && port != "" {
				m[dev] = port
			}
		}
	}
	return m
}

// isWirelessPort reports whether a hardware-port name denotes Wi-Fi.
func isWirelessPort(port string) bool {
	low := strings.ToLower(port)
	return strings.Contains(low, "wi-fi") || strings.Contains(low, "wifi") || strings.Contains(low, "airport")
}
