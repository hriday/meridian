package netroute

import "testing"

func TestParseProcNetRoute(t *testing.T) {
	// Real /proc/net/route shape: default row has Destination 00000000 and a
	// little-endian hex Gateway (0101A8C0 -> 192.168.1.1).
	sample := "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
		"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0\n" +
		"eth0\t0001A8C0\t00000000\t0001\t0\t0\t0\t00FFFFFF\t0\t0\t0\n"
	if got := parseProcNetRoute(sample); got != "192.168.1.1" {
		t.Errorf("parseProcNetRoute = %q, want 192.168.1.1", got)
	}
	// No default route -> empty, never a guess.
	only := "Iface\tDestination\tGateway\tFlags\n" +
		"eth0\t0001A8C0\t00000000\t0001\n"
	if got := parseProcNetRoute(only); got != "" {
		t.Errorf("parseProcNetRoute(no default) = %q, want empty", got)
	}
}

func TestParseRouteGet(t *testing.T) {
	// Real `route -n get default` output on macOS.
	sample := "   route to: default\n" +
		"destination: default\n" +
		"       mask: default\n" +
		"    gateway: 192.168.1.1\n" +
		"  interface: en0\n"
	if got := parseRouteGet(sample); got != "192.168.1.1" {
		t.Errorf("parseRouteGet = %q, want 192.168.1.1", got)
	}
	if got := parseRouteGet("route: writing to routing socket: not in table\n"); got != "" {
		t.Errorf("parseRouteGet(no gateway) = %q, want empty", got)
	}
}
