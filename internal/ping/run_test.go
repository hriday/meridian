package ping

import (
	"context"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakePinger returns scripted (rtt, ok, available) results, one per call.
type fakePinger struct {
	calls   int32
	results []struct {
		rtt       time.Duration
		ok, avail bool
	}
	onCall func(n int)
}

func (f *fakePinger) Ping(ctx context.Context, ip netip.Addr, timeout time.Duration) (time.Duration, bool, bool) {
	n := int(atomic.AddInt32(&f.calls, 1)) - 1
	if f.onCall != nil {
		f.onCall(n)
	}
	if n >= len(f.results) {
		return 0, false, true
	}
	r := f.results[n]
	return r.rtt, r.ok, r.avail
}

func pingN(rtts ...time.Duration) *fakePinger {
	f := &fakePinger{}
	for _, rtt := range rtts {
		f.results = append(f.results, struct {
			rtt       time.Duration
			ok, avail bool
		}{rtt, true, true})
	}
	return f
}

func TestRunAggregates(t *testing.T) {
	f := pingN(10*time.Millisecond, 20*time.Millisecond, 30*time.Millisecond, 20*time.Millisecond, 10*time.Millisecond)
	s, err := Run(context.Background(), f, "127.0.0.1", 5, 0, time.Second, nil)
	require.NoError(t, err)
	require.Equal(t, 5, s.Sent)
	require.Equal(t, 5, s.Received)
	require.Equal(t, "127.0.0.1", s.IP)
	require.True(t, s.Available)
	require.InDelta(t, 10, s.MinMs, 1e-6)
	require.InDelta(t, 30, s.MaxMs, 1e-6)
}

func TestRunStreamsReplies(t *testing.T) {
	f := pingN(10*time.Millisecond, 20*time.Millisecond, 30*time.Millisecond)
	var got []int
	_, err := Run(context.Background(), f, "127.0.0.1", 3, 0, time.Second, func(r Reply) {
		got = append(got, r.Seq)
	})
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, got)
}

func TestRunICMPUnavailable(t *testing.T) {
	f := &fakePinger{results: []struct {
		rtt       time.Duration
		ok, avail bool
	}{{0, false, false}}}
	s, err := Run(context.Background(), f, "127.0.0.1", 5, 0, time.Second, nil)
	require.NoError(t, err)
	require.False(t, s.Available)
	require.Equal(t, 0, s.Sent, "should not loop when ICMP is unavailable")
}

func TestRunCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	f := pingN(10*time.Millisecond, 10*time.Millisecond, 10*time.Millisecond, 10*time.Millisecond, 10*time.Millisecond)
	f.onCall = func(n int) {
		if n == 1 {
			cancel() // cancel after the 2nd ping begins
		}
	}
	s, err := Run(ctx, f, "127.0.0.1", 5, 50*time.Millisecond, time.Second, nil)
	require.NoError(t, err)
	require.LessOrEqual(t, s.Received, 5)
	require.Greater(t, s.Sent, 0)
}
