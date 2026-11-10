package app

import (
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/macvendor"
)

// macLookup is the vendor-lookup function; a package var so tests can inject a
// fake without hitting the network.
var macLookup = macvendor.Lookup

// MACVendor looks up the vendor (IEEE OUI owner) of a MAC address or OUI prefix.
func (a *App) MACVendor(mac string) (macvendor.Result, error) {
	return macLookup(a.lookupCtx(), &http.Client{Timeout: 8 * time.Second}, mac)
}
