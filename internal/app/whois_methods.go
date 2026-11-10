package app

import (
	"time"

	"github.com/hriday/meridian/internal/whois"
)

// whoisQuery is the lookup function; a package var so tests can inject a fake.
var whoisQuery = whois.Query

// Whois queries port-43 Whois for a domain, IP, or ASN and returns the raw
// record plus the chain of servers consulted.
func (a *App) Whois(object string) (whois.Result, error) {
	return whoisQuery(a.lookupCtx(), object, 15*time.Second)
}
