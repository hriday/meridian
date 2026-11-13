package app

import (
	"context"
	"net"
	"time"

	"github.com/hriday/meridian/internal/ptrsweep"
)

// PTRSweep does a reverse-DNS (PTR) lookup across every address in a CIDR and
// returns the ones that resolve to a hostname — a fast map of what's named in a
// subnet. DNS only; no packets to the hosts themselves.
func (a *App) PTRSweep(cidr string) ([]ptrsweep.Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resolver := func(ctx context.Context, ip string) ([]string, error) {
		return net.DefaultResolver.LookupAddr(ctx, ip)
	}
	return ptrsweep.Sweep(ctx, resolver, cidr, 64, nil)
}
