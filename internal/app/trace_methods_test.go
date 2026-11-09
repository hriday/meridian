package app

import (
	"context"
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/trace"
	"github.com/hriday/meridian/internal/tracerun"
	"github.com/stretchr/testify/require"
)

type appFakeProber struct{ reach int }

func (f appFakeProber) Probe(_ context.Context, dst netip.Addr, ttl, _ int, _ time.Duration) (netip.Addr, time.Duration, bool, error) {
	if ttl < f.reach {
		return netip.MustParseAddr(fmt.Sprintf("192.0.2.%d", ttl)), 5 * time.Millisecond, false, nil
	}
	return dst, 6 * time.Millisecond, true, nil
}
func (appFakeProber) Close() error { return nil }

func TestAppTraceroute(t *testing.T) {
	op, oq, ort, orh := tracerun.NewProber, tracerun.Query, tracerun.ResolveTarget, tracerun.ResolveHost
	t.Cleanup(func() {
		tracerun.NewProber, tracerun.Query, tracerun.ResolveTarget, tracerun.ResolveHost = op, oq, ort, orh
	})
	tracerun.NewProber = func(string) (trace.Prober, error) { return appFakeProber{reach: 3}, nil }
	tracerun.Query = func(_ context.Context, _, _, _ string) (dnsx.Result, error) {
		return dnsx.Result{Answers: []dnsx.Record{{Type: "TXT", Value: `"13335 | 1.0.0.0/8 | US | arin | x | CLOUDFLARENET, US"`}}}, nil
	}
	tracerun.ResolveTarget = func(_ context.Context, _ string, _ bool) (netip.Addr, error) {
		return netip.MustParseAddr("1.1.1.1"), nil
	}
	tracerun.ResolveHost = func(_ context.Context, _ string) string { return "" }

	a := New()
	r, err := a.Traceroute("1.1.1.1", false)
	require.NoError(t, err)
	require.True(t, r.Reached)
	require.Len(t, r.Hops, 3)
	require.Equal(t, "13335", r.Hops[2].ASN)
}

func TestTraceCancelGenerations(t *testing.T) {
	a := New()
	aCanceled, bCanceled := false, false
	genA := a.startTrace(func() { aCanceled = true })
	a.startTrace(func() { bCanceled = true }) // starting B cancels A
	require.True(t, aCanceled)

	a.endTrace(genA) // A unwinding must NOT clear B's cancel
	a.StopTrace()    // must still cancel B
	require.True(t, bCanceled)
}
