// Package netroute finds the host's IPv4 default gateway, best-effort and
// read-only. It is deliberately forgiving: when the gateway can't be determined
// or the platform isn't supported, it returns "" rather than an error, so a
// caller shows the gateway only when it's actually known and never a guess.
package netroute

import (
	"encoding/hex"
	"net"
	"strings"
)

// DefaultGateway returns the IPv4 default-gateway address (e.g. "192.168.1.1"),
// or "" when it can't be determined.
func DefaultGateway() string { return defaultGateway() }

// parseProcNetRoute extracts the IPv4 default gateway from the contents of
// Linux's /proc/net/route. Columns are whitespace-separated; the default route
// is the row whose Destination is 00000000 with a non-zero Gateway. Both are
// little-endian hex (so 0101A8C0 is 192.168.1.1).
func parseProcNetRoute(data string) string {
	for _, line := range strings.Split(data, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || f[1] != "00000000" || f[2] == "00000000" {
			continue
		}
		b, err := hex.DecodeString(f[2])
		if err != nil || len(b) != 4 {
			continue
		}
		return net.IPv4(b[3], b[2], b[1], b[0]).String() // little-endian -> dotted
	}
	return ""
}

// parseRouteGet pulls the gateway out of `route -n get default` output (macOS /
// BSD), which contains a line like "    gateway: 192.168.1.1".
func parseRouteGet(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "gateway:"); ok {
			if gw := strings.TrimSpace(rest); net.ParseIP(gw) != nil {
				return gw
			}
		}
	}
	return ""
}
