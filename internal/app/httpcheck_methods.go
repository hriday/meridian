package app

import (
	"time"

	"github.com/hriday/meridian/internal/httpcheck"
)

// HTTPCheck fetches a URL and reports its status, redirect chain, headers,
// protocol, TLS version, and per-phase timing.
func (a *App) HTTPCheck(rawURL string) (httpcheck.Result, error) {
	return httpcheck.Check(a.lookupCtx(), rawURL, 15*time.Second)
}
