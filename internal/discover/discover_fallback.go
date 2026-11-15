//go:build !darwin

package discover

import (
	"context"
	"time"
)

// systemMDNS on non-macOS platforms uses the raw multicast probe. (macOS has a
// separate cgo backend that talks to the system mDNSResponder, because there the
// responder owns port 5353 exclusively and no other socket can receive the LAN's
// multicast answers.)
func systemMDNS(ctx context.Context, window time.Duration) ([]Service, error) {
	return queryMDNS(ctx, window)
}
