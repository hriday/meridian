package app

import (
	"context"
	"runtime"
	"time"

	"github.com/hriday/meridian/internal/ndp"
)

// NDPScan reads the IPv6 neighbor cache and, when withRA is set and the platform
// supports it, solicits and parses Router Advertisements. RA listening needs a
// raw ICMPv6 socket (root), so it may return a note instead of routers when the
// app isn't privileged.
func (a *App) NDPScan(withRA bool) (ndp.Result, error) {
	res := ndp.Result{OS: runtime.GOOS}
	ns, err := ndp.Neighbors(a.lookupCtx())
	if err != nil {
		return res, err
	}
	res.Neighbors = ns
	res.DefaultRouters, _ = ndp.DefaultRouters(a.lookupCtx()) // unprivileged, best-effort
	if withRA {
		if !ndp.RASupported() {
			res.RANote = "Router Advertisement listening isn't available on this OS."
			return res, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ras, rerr := ndp.ListenRA(ctx, "", 4*time.Second)
		if rerr != nil {
			res.RANote = rerr.Error()
		} else {
			res.Routers = ras
		}
	}
	return res, nil
}
