package app

import (
	"github.com/hriday/meridian/internal/axfr"
	"github.com/hriday/meridian/internal/sysconf"
)

// AXFRTest checks whether domain's authoritative nameservers allow an
// unauthenticated zone transfer. resolver (empty = system) is used only to find
// the nameservers; the transfer is attempted against each of them directly.
func (a *App) AXFRTest(domain, resolver string) (axfr.Report, error) {
	if resolver == "" {
		if f := sysconf.Resolvers(); len(f.Values) > 0 {
			resolver = f.Values[0]
		} else {
			resolver = "1.1.1.1"
		}
	}
	return axfr.Test(a.lookupCtx(), domain, resolver)
}
