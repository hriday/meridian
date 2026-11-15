//go:build !darwin && !linux && !windows

package discover

import (
	"context"
	"time"
)

// systemMDNS on platforms without a native backend uses the raw multicast probe.
// (macOS, Linux, and Windows each have their own backend that talks to the
// system mDNS resolver — mDNSResponder, Avahi, and DnsServiceBrowse respectively
// — which hold the aggregated LAN view a raw socket can't reliably get.)
func systemMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	return queryMDNS(ctx, window)
}
