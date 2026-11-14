package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/rtp"
)

// RTPSupported reports whether live RTP capture works on this OS.
func (a *App) RTPSupported() bool { return rtp.Supported() }

// RTPCapture sniffs the egress interface for seconds and returns the RTP streams
// seen, with per-stream jitter / loss / codec. Needs raw-capture privileges.
func (a *App) RTPCapture(seconds int) ([]rtp.Stream, error) {
	if seconds <= 0 {
		seconds = int(rtp.DefaultWindow / time.Second)
	}
	ctx, cancel := context.WithTimeout(a.lookupCtx(), time.Duration(seconds+5)*time.Second)
	defer cancel()
	return rtp.Capture(ctx, "", time.Duration(seconds)*time.Second)
}
