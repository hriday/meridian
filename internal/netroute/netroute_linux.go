//go:build linux

package netroute

import "os"

func defaultGateway() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	return parseProcNetRoute(string(data))
}
