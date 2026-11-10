package app

import (
	"time"

	"github.com/hriday/meridian/internal/ping"
	"github.com/hriday/meridian/internal/scan"
)

// newICMPPinger builds the ICMP pinger for PingHost; a package var so tests can
// inject a fake. (Distinct from the App.newPinger field used by the sweep.)
var newICMPPinger = func() ping.Pinger { return scan.NewICMPPinger() }

// maxPingCount bounds a single PingHost call. The method is synchronous (the
// whole run blocks the bound call at 500ms/ping), so an unbounded count would
// freeze the UI; 30 pings is ~15s, the ceiling we allow from the GUI.
const maxPingCount = 30

// clampCount applies the default (5) and the ceiling.
func clampCount(count int) int {
	if count <= 0 {
		return 5
	}
	if count > maxPingCount {
		return maxPingCount
	}
	return count
}

// PingHost pings host count times (default 5, capped at maxPingCount) and
// returns latency statistics (min/avg/max/stddev/jitter) and packet loss. It
// runs synchronously at a humane interval; if unprivileged ICMP isn't
// available, the result says so.
func (a *App) PingHost(host string, count int) (ping.Stats, error) {
	return ping.Run(a.lookupCtx(), newICMPPinger(), host, clampCount(count), 500*time.Millisecond, time.Second, nil)
}
