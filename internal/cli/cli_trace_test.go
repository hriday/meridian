package cli

import (
	"bytes"
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

type traceFakeProber struct{ reach int }

func (f traceFakeProber) Probe(_ context.Context, dst netip.Addr, ttl, _ int, _ time.Duration) (netip.Addr, time.Duration, bool, error) {
	if ttl < f.reach {
		return netip.MustParseAddr(fmt.Sprintf("192.0.2.%d", ttl)), 5 * time.Millisecond, false, nil
	}
	return dst, 6 * time.Millisecond, true, nil
}
func (traceFakeProber) Close() error { return nil }

func withFakeTrace(t *testing.T, newProber func(string) (trace.Prober, error)) {
	op, oq, ort, orh := tracerun.NewProber, tracerun.Query, tracerun.ResolveTarget, tracerun.ResolveHost
	t.Cleanup(func() {
		tracerun.NewProber, tracerun.Query, tracerun.ResolveTarget, tracerun.ResolveHost = op, oq, ort, orh
	})
	tracerun.NewProber = newProber
	tracerun.Query = func(_ context.Context, _, _, _ string) (dnsx.Result, error) {
		return dnsx.Result{Answers: []dnsx.Record{{Type: "TXT", Value: `"13335 | 1.0.0.0/8 | US | arin | x | CLOUDFLARENET, US"`}}}, nil
	}
	tracerun.ResolveTarget = func(_ context.Context, _ string, _ bool) (netip.Addr, error) {
		return netip.MustParseAddr("1.1.1.1"), nil
	}
	tracerun.ResolveHost = func(_ context.Context, _ string) string { return "" }
}

func TestRunTraceJSON(t *testing.T) {
	withFakeTrace(t, func(string) (trace.Prober, error) { return traceFakeProber{reach: 3}, nil })
	var out, errb bytes.Buffer
	code := Run([]string{"trace", "1.1.1.1", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), `"reached":true`)
	require.Contains(t, out.String(), "13335")
}

func TestRunTraceSocketErrorSurfaced(t *testing.T) {
	withFakeTrace(t, func(string) (trace.Prober, error) {
		return nil, fmt.Errorf("couldn't open an ICMP socket for ip4")
	})
	var out, errb bytes.Buffer
	code := Run([]string{"trace", "1.1.1.1"}, &out, &errb)
	require.NotEqual(t, 0, code)
	require.Contains(t, errb.String(), "ICMP")
}
