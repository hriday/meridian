package app

import (
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/cve"
)

func cveClient() *http.Client { return &http.Client{Timeout: 25 * time.Second} }

// CVELookup resolves a single CVE by id from the NVD (free, keyless).
func (a *App) CVELookup(id string) (cve.CVE, error) {
	return cve.Lookup(a.lookupCtx(), cveClient(), id)
}

// CVESearch returns CVEs matching a keyword from the NVD (free, keyless).
func (a *App) CVESearch(keyword string) ([]cve.CVE, error) {
	return cve.Search(a.lookupCtx(), cveClient(), keyword)
}

// CVEIsID reports whether a query looks like a CVE identifier (so the UI knows
// whether to do a direct lookup or a keyword search).
func (a *App) CVEIsID(s string) bool { return cve.IsID(s) }
