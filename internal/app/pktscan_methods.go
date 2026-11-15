package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/pktscan"
)

// PacketScanSupported reports whether live packet capture works on this OS.
func (a *App) PacketScanSupported() bool { return pktscan.Supported() }

// PacketScan captures on the egress interface for seconds and returns a traffic
// summary — top talkers, protocol mix, TCP resets/retransmits, DNS errors, and
// TLS SNIs. Needs raw-capture privileges (sudo / ChmodBPF).
func (a *App) PacketScan(seconds int) (pktscan.Summary, error) {
	if seconds <= 0 {
		seconds = int(pktscan.DefaultWindow / time.Second)
	}
	ctx, cancel := context.WithTimeout(a.lookupCtx(), time.Duration(seconds+5)*time.Second)
	defer cancel()
	return pktscan.Run(ctx, "", time.Duration(seconds)*time.Second, 10)
}
