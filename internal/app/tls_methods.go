package app

import (
	"time"

	"github.com/hriday/meridian/internal/tlscheck"
)

// TLSCheck connects to host[:port] (default 443) and returns its TLS
// certificate with a plain-language health verdict.
func (a *App) TLSCheck(target string) (tlscheck.CertReport, error) {
	return tlscheck.Check(a.lookupCtx(), target, 8*time.Second)
}
