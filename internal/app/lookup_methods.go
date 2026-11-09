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

// LookupPrefix resolves a CIDR block (origin AS, RPKI, registry, related prefixes).
func (a *App) LookupPrefix(prefix string) (lookup.PrefixReport, error) {
	return a.lookup.LookupPrefix(a.lookupCtx(), prefix)
}

// LookupDomain resolves a domain (DNS records, hosting AS, RDAP registration).
func (a *App) LookupDomain(name string) (lookup.DomainReport, error) {
	return a.lookup.LookupDomain(a.lookupCtx(), name)
}
