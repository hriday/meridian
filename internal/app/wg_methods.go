package app

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/hriday/meridian/internal/wgprobe"
)

// WgProbe checks a UDP endpoint for a WireGuard responder (default port 51820)
// by sending a Noise handshake initiation. With the peer's base64 public key the
// initiation carries a valid mac1, so a cookie or handshake reply positively
// confirms WireGuard; without it, WireGuard's silence is expected and
// inconclusive. Read-only — it never completes a handshake.
func (a *App) WgProbe(host string, port int, peerPubKey string) (wgprobe.Result, error) {
	if port == 0 {
		port = 51820
	}
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 8*time.Second)
	defer cancel()
	return wgprobe.Probe(ctx, net.JoinHostPort(host, strconv.Itoa(port)), peerPubKey)
}
