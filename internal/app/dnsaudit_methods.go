package app

import (
	"time"

	"github.com/hriday/meridian/internal/dnsaudit"
	"github.com/hriday/meridian/internal/sysconf"
)

// DNSAudit grades domain's DNS/email posture (NS, MX, SPF, DMARC, DNSSEC),
// querying via resolver. An empty resolver defaults to the host's system
// resolver.
func (a *App) DNSAudit(domain, resolver string) (dnsaudit.Report, error) {
	if resolver == "" {
		if f := sysconf.Resolvers(); len(f.Values) > 0 {
			resolver = f.Values[0]
		}
	}
	return dnsaudit.Audit(a.lookupCtx(), domain, resolver, 8*time.Second)
}
