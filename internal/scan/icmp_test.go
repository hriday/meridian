package scan

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestICMPPingerAvailability(t *testing.T) {
	var _ Pinger = NewICMPPinger() // satisfies the seam

	p := NewICMPPinger()
	ip := netip.MustParseAddr("127.0.0.1")
	// Must not panic, and availability must be stable across calls (real ICMP
	// reachability isn't asserted — unprivileged ICMP may be disallowed in CI).
	_, _, a1 := p.Ping(context.Background(), ip, 500*time.Millisecond)
	_, _, a2 := p.Ping(context.Background(), ip, 500*time.Millisecond)
	require.Equal(t, a1, a2)
}
