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

func TestUniqueNetworks(t *testing.T) {
	hops := []trace.Hop{
		{ASN: "7922", ASOwner: "Comcast"},
		{ASN: "7922", ASOwner: "Comcast"}, // dup collapses
		{ASN: ""},                         // no AS -> skipped
		{ASN: "13335", ASOwner: "Cloudflare"},
		{ASN: "13335"},
	}
	got := uniqueNetworks(hops)
	require.Len(t, got, 2)
	require.Equal(t, "AS7922", got[0].ASN)
	require.Equal(t, "Comcast", got[0].Owner)
	require.Equal(t, "AS13335", got[1].ASN)
}

// Run must hand the path's networks to EnrichNetworks and attach the result.
func TestRunCallsEnrichNetworks(t *testing.T) {
	op, orh, oe := NewProber, ResolveHost, EnrichNetworks
	t.Cleanup(func() { NewProber, ResolveHost, EnrichNetworks = op, orh, oe })
	NewProber = func(string) (trace.Prober, error) { return nopProber{}, nil }
	ResolveHost = func(context.Context, string) string { return "" }
	called := false
	EnrichNetworks = func(ctx context.Context, nets []trace.NetInfo) []trace.NetInfo {
		called = true
		return []trace.NetInfo{{ASN: "AS13335", Owner: "Cloudflare", Policy: "open"}}
	}
	r, err := Run(context.Background(), "1.1.1.1", false, nil)
	require.NoError(t, err)
	require.True(t, called)
	require.Len(t, r.Networks, 1)
	require.Equal(t, "open", r.Networks[0].Policy)
}
