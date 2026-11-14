package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/mtu"
)

// MTUSupported reports whether path-MTU discovery works on this OS (macOS /
// Linux). The frontend uses it to show an unsupported notice up front.
func (a *App) MTUSupported() bool { return mtu.Supported() }

// PathMTU discovers the path MTU to target. It self-checks that the OS enforces
// Don't-Fragment; if not (some VPN / Network-Extension setups, some macOS
// betas) the Result says so rather than reporting a bogus number.
func (a *App) PathMTU(target string) (mtu.Result, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 30*time.Second)
	defer cancel()
	return mtu.DiscoverHost(ctx, mtu.Options{Target: target})
}
