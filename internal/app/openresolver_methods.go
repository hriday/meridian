package app

import (
	"github.com/hriday/meridian/internal/openresolver"
)

// OpenResolverTest probes resolver (ip or host) and reports whether it recurses
// for arbitrary clients — an accidental open resolver and DNS-amplification risk.
func (a *App) OpenResolverTest(resolver string) (openresolver.Report, error) {
	return openresolver.Test(a.lookupCtx(), resolver)
}
