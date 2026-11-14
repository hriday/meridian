package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/dnsprop"
	"github.com/hriday/meridian/internal/dnsx"
)

// dnsPropQuery is the per-resolver query seam (tests inject a fake).
var dnsPropQuery = dnsx.Query

// DNSPropagation queries name's rtype record across a spread of global public
// resolvers at once and reports whether they agree — i.e. how far a change has
// propagated.
func (a *App) DNSPropagation(name, rtype string) (dnsprop.Report, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 12*time.Second)
	defer cancel()
	return dnsprop.Check(ctx, dnsPropQuery, name, rtype, dnsprop.Public()), nil
}
