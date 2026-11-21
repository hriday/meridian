package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/linkpath"
)

// LinkInspect inspects the link toward the router: reads the egress interface's
// speed/duplex (a hub tell) and listens for a switch's LLDP/CDP advertisement,
// then returns an honest, hedged verdict about whether a hub or switch sits
// between this machine and the gateway. The duplex read needs no privilege; the
// LLDP capture does, and its absence is reported rather than hidden.
// User-initiated. seconds <= 0 uses the default listen window.
func (a *App) LinkInspect(seconds int) (linkpath.Report, error) {
	window := time.Duration(seconds) * time.Second
	if window <= 0 {
		window = linkpath.DefaultLLDPWindow
	}
	ctx, cancel := context.WithTimeout(context.Background(), window+10*time.Second)
	defer cancel()
	return linkpath.Inspect(ctx, window)
}
