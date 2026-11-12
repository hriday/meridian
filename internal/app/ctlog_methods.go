package app

import (
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/ctlog"
)

// CTSubdomains discovers a domain's subdomains from public Certificate
// Transparency logs (crt.sh) — a free, keyless map of the attack surface. It
// finds only hosts that have had a TLS certificate. User-initiated.
func (a *App) CTSubdomains(domain string) ([]ctlog.Subdomain, error) {
	// crt.sh can be slow on large domains, so allow a generous timeout.
	client := &http.Client{Timeout: 45 * time.Second}
	return ctlog.Subdomains(a.lookupCtx(), client, domain)
}
