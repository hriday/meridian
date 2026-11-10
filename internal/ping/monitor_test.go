package ping

import (
	"context"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type monFake struct {
	ok    bool
	avail bool
}

func (f monFake) Ping(_ context.Context, _ netip.Addr, _ time.Duration) (time.Duration, bool, bool) {
	return 5 * time.Millisecond, f.ok, f.avail
}

func TestMonitorStreamsUntilCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var n int64
	done := make(chan struct{})
	go func() {
		avail, err := Monitor(ctx, monFake{ok: true, avail: true}, "10.0.0.1", time.Millisecond, time.Millisecond, func(r Reply) {
			if atomic.AddInt64(&n, 1) == 3 {
				cancel() // stop after a few
			}
		})
		require.NoError(t, err)
		require.True(t, avail)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Monitor did not stop on cancel")
	}
	require.GreaterOrEqual(t, atomic.LoadInt64(&n), int64(3))
}

func TestMonitorUnavailable(t *testing.T) {
	calls := 0
	avail, err := Monitor(context.Background(), monFake{ok: false, avail: false}, "10.0.0.1", time.Millisecond, time.Millisecond, func(Reply) { calls++ })
	require.NoError(t, err)
	require.False(t, avail)
	require.Equal(t, 0, calls) // returns before streaming
}

func TestMonitorResolveError(t *testing.T) {
	_, err := Monitor(context.Background(), monFake{ok: true, avail: true}, "definitely not a host!!", time.Millisecond, time.Millisecond, nil)
	require.Error(t, err)
}
