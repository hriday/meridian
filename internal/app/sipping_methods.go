package app

import (
	"context"
	"net"
	"time"

	"github.com/hriday/meridian/internal/sipping"
)

// SIPProbe sends a SIP OPTIONS to host over transport (udp/tcp/tls) and reports
// whether it answers and what it supports — the canonical SIP reachability test.
func (a *App) SIPProbe(host, port, transport string) (sipping.Result, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 10*time.Second)
	defer cancel()
	if port == "" {
		port = "5060"
		if transport == "tls" {
			port = "5061"
		}
	}
	d := &net.Dialer{}
	return sipping.Probe(ctx, d.DialContext, host, port, transport)
}
