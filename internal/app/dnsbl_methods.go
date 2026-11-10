package app

import (
	"time"

	"github.com/hriday/meridian/internal/dnsbl"
	"github.com/hriday/meridian/internal/sysconf"
)

// Blocklist checks whether ip is listed on the default public blocklists,
// querying via resolver. An empty resolver defaults to the host's system
// resolver (big public resolvers get blocklist queries refused, so the user's
// own resolver is the right default).
func (a *App) Blocklist(ip, resolver string) (dnsbl.Report, error) {
	if resolver == "" {
		if f := sysconf.Resolvers(); len(f.Values) > 0 {
			resolver = f.Values[0]
		}
	}
	return dnsbl.Check(a.lookupCtx(), ip, resolver, dnsbl.DefaultZones(), 8*time.Second)
}
