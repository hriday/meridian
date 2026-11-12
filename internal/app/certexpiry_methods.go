package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/certexpiry"
)

// CertExpiryCheck takes a free-form list of hosts (host, host:port, or a URL,
// one per line or comma/space separated), dials each over TLS, and reports how
// soon its certificate expires — soonest first. Direct TLS dials only; no
// third-party service. Hosts that can't be reached are listed with the error.
func (a *App) CertExpiryCheck(input string) []certexpiry.Result {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hosts := certexpiry.SplitHosts(input)
	return certexpiry.Check(ctx, certexpiry.Dial, hosts, time.Now())
}
