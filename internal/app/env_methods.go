package app

import (
	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/ntp"
	"github.com/hriday/meridian/internal/sysconf"
)

// Query functions behind package vars so tests inject fakes.
var (
	ntpQueryFn = ntp.Query
	dnsQueryFn = dnsx.Query
)

// NTPQuery queries a time server for stratum, offset, and delay.
func (a *App) NTPQuery(server string) (ntp.Reply, error) {
	return ntpQueryFn(a.lookupCtx(), server)
}

// DNSQuery resolves name's rtype records against server (empty = caller picks).
func (a *App) DNSQuery(name, rtype, server string) (dnsx.Result, error) {
	return dnsQueryFn(a.lookupCtx(), name, rtype, server)
}

// NTPSystem reports the host's configured time server(s), best-effort.
func (a *App) NTPSystem() sysconf.Found { return sysconf.TimeServers() }

// DNSSystem reports the host's configured DNS resolver(s), best-effort.
func (a *App) DNSSystem() sysconf.Found { return sysconf.Resolvers() }
