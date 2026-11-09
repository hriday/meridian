package tracerun

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/trace"
	"github.com/stretchr/testify/require"
)

type nopProber struct{}

func (nopProber) Probe(context.Context, netip.Addr, int, int, time.Duration) (netip.Addr, time.Duration, bool, error) {
	return netip.Addr{}, 0, false, nil
}
func (nopProber) Close() error { return nil }

// A user-initiated Stop cancels the context; Run must return the partial
// Result with a nil error (a clean stop), not surface context.Canceled.
func TestRunSwallowsCancel(t *testing.T) {
	op, orh := NewProber, ResolveHost
	t.Cleanup(func() { NewProber, ResolveHost = op, orh })
	NewProber = func(string) (trace.Prober, error) { return nopProber{}, nil }
	ResolveHost = func(context.Context, string) string { return "" }

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := Run(ctx, "1.1.1.1", false, nil)
	require.NoError(t, err)
	require.False(t, r.Reached)
	require.Equal(t, "1.1.1.1", r.Target)
}
