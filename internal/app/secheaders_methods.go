package app

import (
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/secheaders"
)

// SecurityHeaders fetches a URL and grades its live HTTP security headers,
// mapping each finding to its OWASP Top 10 category. User-initiated.
func (a *App) SecurityHeaders(url string) (secheaders.Report, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	return secheaders.Scan(a.lookupCtx(), client, url)
}
