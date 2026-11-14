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
		seconds = 3
	}
	if seconds > 15 {
		seconds = 15
	}
	window := time.Duration(seconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), window+3*time.Second)
	defer cancel()
	return discover.Run(ctx, window)
}
