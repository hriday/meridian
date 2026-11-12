package app

import (
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/hibp"
)

func hibpClient() *http.Client { return &http.Client{Timeout: 20 * time.Second} }

// HIBPBreaches returns the full Have I Been Pwned breach catalogue (free, no key)
// — which sites were breached and what data was exposed. The UI filters it by
// name/domain locally. User-initiated; no background calls.
func (a *App) HIBPBreaches() ([]hibp.Breach, error) {
	return hibp.Breaches(a.lookupCtx(), hibpClient())
}

// HIBPLatestBreach returns the most recently added breach (free, no key).
func (a *App) HIBPLatestBreach() (hibp.Breach, error) {
	return hibp.LatestBreach(a.lookupCtx(), hibpClient())
}
