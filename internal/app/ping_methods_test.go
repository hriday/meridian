package app

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/ping"
	"github.com/stretchr/testify/require"
)

type okPinger struct{}

func (okPinger) Ping(_ context.Context, _ netip.Addr, _ time.Duration) (time.Duration, bool, bool) {
	return 12 * time.Millisecond, true, true
}

func TestPingHostMethod(t *testing.T) {
	old := newICMPPinger
	newICMPPinger = func() ping.Pinger { return okPinger{} }
	t.Cleanup(func() { newICMPPinger = old })

	s, err := New().PingHost("127.0.0.1", 3)
	require.NoError(t, err)
	require.Equal(t, 3, s.Received)
	require.Equal(t, "127.0.0.1", s.IP)
	require.True(t, s.Available)
	require.InDelta(t, 12, s.AvgMs, 1e-6)
}
