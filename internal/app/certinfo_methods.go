package app

import (
	"time"

	"github.com/hriday/meridian/internal/certinfo"
)

// CertDecode decodes pasted PEM certificates and CSRs into a readable summary,
// locally — the cert/CSR never leaves the machine.
func (a *App) CertDecode(pemText string) (certinfo.Result, error) {
	return certinfo.Decode(pemText, time.Now())
}
