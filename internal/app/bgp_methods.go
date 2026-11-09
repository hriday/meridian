package app

import "github.com/hriday/meridian/internal/lookup"

// LookupBGP returns the control-plane view of a prefix or IP: origin AS(es),
// AS paths, upstreams/transit, visibility, RPKI, and the collectors that see it.
func (a *App) LookupBGP(input string) (lookup.BGPReport, error) {
	return a.lookup.LookupBGP(a.lookupCtx(), input)
}
