package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/ikeprobe"
)

// IkeProbe checks an IKE/IPsec VPN responder (IKEv2, UDP/500) by sending a real
// IKE_SA_INIT: whether it answers, that it's IKEv2, the proposal it accepts
// (encryption / PRF / integrity / DH), and whether NAT traversal is supported.
// Read-only — it never completes the handshake.
func (a *App) IkeProbe(host string) (ikeprobe.Result, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 8*time.Second)
	defer cancel()
	return ikeprobe.Probe(ctx, host)
}
