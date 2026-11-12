package app

import (
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/hibp"
	"github.com/hriday/meridian/internal/xon"
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

// EmailBreachCheck checks an email address against XposedOrNot's free, keyless
// breach API and returns the breaches it appears in. Unlike the password check,
// the email address IS sent to the service — the UI makes that clear. This is a
// user-initiated call; nothing runs automatically.
func (a *App) EmailBreachCheck(email string) ([]string, error) {
	return xon.CheckEmail(a.lookupCtx(), hibpClient(), email)
}
