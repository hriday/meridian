package app

import (
	"time"

	"github.com/hriday/meridian/internal/ping"
	"github.com/hriday/meridian/internal/scan"
)

// newICMPPinger builds the ICMP pinger for PingHost; a package var so tests can
// inject a fake. (Distinct from the App.newPinger field used by the sweep.)
var newICMPPinger = func() ping.Pinger { return scan.NewICMPPinger() }

// PingHost pings host count times (default 5) and returns latency statistics
// (min/avg/max/stddev/jitter) and packet loss. It runs synchronously at a
// humane interval; if unprivileged ICMP isn't available, the result says so.
func (a *App) PingHost(host string, count int) (ping.Stats, error) {
	if count <= 0 {
		count = 5
	}
	return ping.Run(a.lookupCtx(), newICMPPinger(), host, count, 500*time.Millisecond, time.Second, nil)
}
