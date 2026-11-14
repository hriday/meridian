package app

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/stun"
)

// Two distinct public STUN servers (different destinations), so a differing
// mapping reveals a symmetric NAT. Cloudflare first — it's widely reachable;
// some networks filter Google's high STUN port.
var defaultSTUN = []string{"stun.cloudflare.com:3478", "stun.l.google.com:19302"}

// STUNDiscover finds this host's public (reflexive) address and reasons about
// the NAT in front of it. An empty server uses public defaults; a provided one
// is paired with the defaults so the mapping type can still be classified.
func (a *App) STUNDiscover(server string) (stun.Result, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 8*time.Second)
	defer cancel()
	servers := defaultSTUN
	if s := strings.TrimSpace(server); s != "" {
		servers = append([]string{withSTUNPort(s)}, defaultSTUN...)
	}
	return stun.Discover(ctx, servers)
}

func withSTUNPort(s string) string {
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s
	}
	return net.JoinHostPort(s, "3478")
}
