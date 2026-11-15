package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/discover"
)

// DiscoverLAN finds local-network devices by what they advertise — mDNS/Bonjour
// (printers, Chromecasts, AirPlay, NAS, …) and SSDP/UPnP (routers, smart TVs,
// media servers, IoT). Multicast UDP, no privileges, point-in-time. seconds is
// the listening window, clamped to a sane range.
func (a *App) DiscoverLAN(seconds int) (discover.Result, error) {
	if seconds < 1 {
		seconds = 8
	}
	if seconds > 25 {
		seconds = 25
	}
	window := time.Duration(seconds) * time.Second
	// mDNS holds one socket open for the whole window (a cold unicast answer can
	// take ~4-5s); give the context a little headroom beyond it.
	ctx, cancel := context.WithTimeout(context.Background(), window+5*time.Second)
	defer cancel()
	return discover.Run(ctx, window)
}
