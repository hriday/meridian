package app

import "github.com/hriday/meridian/internal/lookup"

// LookupIP resolves an IP address to its owner (routing + registry).
func (a *App) LookupIP(ip string) (lookup.IPReport, error) {
	return a.lookup.LookupIP(a.lookupCtx(), ip)
}

// LookupASN resolves an ASN to its holder, routing counts, and registry.
func (a *App) LookupASN(asn string) (lookup.ASNReport, error) {
	return a.lookup.LookupASN(a.lookupCtx(), asn)
}
