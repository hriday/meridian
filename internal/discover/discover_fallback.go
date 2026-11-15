//go:build !darwin && !linux

package discover

import (
	"context"
	"time"
)

// systemMDNS on platforms without a native backend uses the raw multicast probe.
// (macOS and Linux have their own backends that talk to the system mDNS resolver
// — mDNSResponder and Avahi respectively — which hold the aggregated LAN view a
// raw socket can't reliably get.)
func systemMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	return queryMDNS(ctx, window)
}
