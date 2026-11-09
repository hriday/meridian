package trace

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type hopStep struct {
	addr    netip.Addr
	reached bool
}

// fakeProber returns a scripted responder per TTL; a missing TTL is a timeout.
type fakeProber struct{ path map[int]hopStep }

func (f *fakeProber) Probe(_ context.Context, _ netip.Addr, ttl, _ int, _ time.Duration) (netip.Addr, time.Duration, bool, error) {
	s, ok := f.path[ttl]
	if !ok {
		return netip.Addr{}, 0, false, nil // timeout
	}
	return s.addr, 5 * time.Millisecond, s.reached, nil
}
func (f *fakeProber) Close() error { return nil }

func addr(s string) netip.Addr { return netip.MustParseAddr(s) }

func TestRunReachesDestination(t *testing.T) {
	dst := addr("104.16.123.45")
	p := &fakeProber{path: map[int]hopStep{
		1: {addr("192.168.1.1"), false},
		2: {addr("96.120.5.1"), false},
		3: {dst, true},
	}}
	var fired int
	r, err := Run(context.Background(), dst, p, Options{MaxHops: 30, Probes: 3}, func(Hop) { fired++ })
	require.NoError(t, err)
	require.True(t, r.Reached)
	require.Len(t, r.Hops, 3)
	require.Equal(t, 3, fired)
	require.Len(t, r.Hops[0].RTTs, 3)
	require.Equal(t, "104.16.123.45", r.Hops[2].Addr)
}

func TestRunNeverReaches(t *testing.T) {
	dst := addr("104.16.123.45")
	p := &fakeProber{path: map[int]hopStep{}} // everything times out
	r, err := Run(context.Background(), dst, p, Options{MaxHops: 5, Probes: 2}, func(Hop) {})
	require.NoError(t, err)
	require.False(t, r.Reached)
	require.Len(t, r.Hops, 5)
	require.True(t, r.Hops[0].Timeout)
}

func TestRunMidPathTimeouts(t *testing.T) {
	dst := addr("104.16.123.45")
	p := &fakeProber{path: map[int]hopStep{
		1: {addr("192.168.1.1"), false},
		3: {dst, true}, // ttl 2 missing -> a timeout hop in the middle
	}}
	r, err := Run(context.Background(), dst, p, Options{MaxHops: 30, Probes: 2}, func(Hop) {})
	require.NoError(t, err)
	require.True(t, r.Reached)
	require.Len(t, r.Hops, 3)
	require.True(t, r.Hops[1].Timeout)
	require.False(t, r.Hops[1].Addr != "")
}

func TestRunCancel(t *testing.T) {
	dst := addr("104.16.123.45")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Run(ctx, dst, &fakeProber{path: map[int]hopStep{}}, Options{MaxHops: 30, Probes: 3}, func(Hop) {})
	require.Error(t, err)
}
