package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/h3probe"
)

// HTTP3Probe tests HTTP/3 (QUIC) reachability to host:port (port 0 → 443): the
// real QUIC handshake plus whether the server advertises h3 via Alt-Svc, so you
// can tell "no h3 support" from "h3 blocked on your path".
func (a *App) HTTP3Probe(host string, port int) h3probe.Result {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 25*time.Second)
	defer cancel()
	return h3probe.Probe(ctx, host, port)
}
