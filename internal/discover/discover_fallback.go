//go:build !darwin && !windows

package discover

import (
	"context"
	"time"
)

// systemMDNS on platforms without a native backend uses the raw multicast probe.
// (macOS and Windows have their own backends that talk to the system mDNS
// resolver; Linux gains an Avahi backend in a separate change.)
func systemMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	return queryMDNS(ctx, window)
}
