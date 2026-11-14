package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/lldp"
)

// LLDPSupported reports whether LLDP/CDP capture works on this OS (macOS/Linux).
func (a *App) LLDPSupported() bool { return lldp.Supported() }

// LLDPDiscover listens for LLDP and CDP advertisements and returns the switch
// neighbors heard ("what am I plugged into"). It needs elevated privileges (raw
// capture); without them it returns an error saying so. It blocks for up to the
// default listen window.
func (a *App) LLDPDiscover() ([]lldp.Neighbor, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), lldp.DefaultWindow+5*time.Second)
	defer cancel()
	return lldp.Listen(ctx, "", lldp.DefaultWindow)
}
