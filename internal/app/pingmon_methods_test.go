package app

import (
	"context"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/metrics"
	"github.com/hriday/meridian/internal/scan"
	"github.com/stretchr/testify/require"
)

type scriptPinger struct {
	rttNanos *int64
	ok       bool
	avail    bool
}

func (p scriptPinger) Ping(_ context.Context, _ netip.Addr, _ time.Duration) (time.Duration, bool, bool) {
	var d time.Duration
	if p.rttNanos != nil {
		d = time.Duration(atomic.LoadInt64(p.rttNanos))
	}
	return d, p.ok, p.avail
}

type ev struct {
	name string
	snap metrics.Snapshot
}

func capture(t *testing.T) func() []ev {
	t.Helper()
	var mu sync.Mutex
	var evs []ev
	old := emitEvent
	emitEvent = func(_ context.Context, name string, data ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if len(data) == 1 {
			if s, ok := data[0].(metrics.Snapshot); ok {
				evs = append(evs, ev{name, s})
			}
		}
	}
	oldInt := pingInterval
	pingInterval = time.Millisecond
	t.Cleanup(func() { emitEvent = old; pingInterval = oldInt })
	return func() []ev { mu.Lock(); defer mu.Unlock(); return append([]ev(nil), evs...) }
}

func steadyPinger(a *App, rtt time.Duration) {
	n := int64(rtt)
	a.newPinger = func() scan.Pinger { return scriptPinger{rttNanos: &n, ok: true, avail: true} }
}

func TestStartEmitsSnapshots(t *testing.T) {
	a := New()
	get := capture(t)
	steadyPinger(a, 10*time.Millisecond)
	require.NoError(t, a.StartPingMonitor("s1", "10.0.0.1"))
	require.Eventually(t, func() bool { return len(get()) >= 3 }, 2*time.Second, 5*time.Millisecond)
	a.StopPingMonitor("s1")
	evs := get()
	require.GreaterOrEqual(t, len(evs), 3)
	require.Equal(t, "metrics:s1", evs[0].name)
}

func TestStopNoLeak(t *testing.T) {
	a := New()
	get := capture(t)
	steadyPinger(a, 10*time.Millisecond)
	require.NoError(t, a.StartPingMonitor("s1", "10.0.0.1"))
	require.Eventually(t, func() bool { return len(get()) >= 2 }, 2*time.Second, 5*time.Millisecond)
	a.StopPingMonitor("s1") // blocks until the goroutine exits
	before := len(get())
	time.Sleep(30 * time.Millisecond) // many intervals
	require.Equal(t, before, len(get()), "no events after Stop")
}

func TestUnavailableSurfaces(t *testing.T) {
	a := New()
	get := capture(t)
	a.newPinger = func() scan.Pinger { return scriptPinger{ok: false, avail: false} }
	require.NoError(t, a.StartPingMonitor("s1", "10.0.0.1"))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if e.snap.State == "error" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	a.StopPingMonitor("s1")
}

func TestSetBaselineAnomaly(t *testing.T) {
	a := New()
	get := capture(t)
	n := int64(10 * time.Millisecond)
	a.newPinger = func() scan.Pinger { return scriptPinger{rttNanos: &n, ok: true, avail: true} }
	require.NoError(t, a.StartPingMonitor("s1", "10.0.0.1"))
	require.Eventually(t, func() bool { return len(get()) >= 12 }, 2*time.Second, 5*time.Millisecond)
	a.SetPingBaseline("s1")
	atomic.StoreInt64(&n, int64(30*time.Millisecond)) // jump well out of the frozen band
	require.Eventually(t, func() bool {
		evs := get()
		return len(evs) > 0 && evs[len(evs)-1].snap.State == "anomaly"
	}, 2*time.Second, 5*time.Millisecond)
	a.StopPingMonitor("s1")
}

func TestTwoStreamsIndependent(t *testing.T) {
	a := New()
	get := capture(t)
	steadyPinger(a, 10*time.Millisecond)
	require.NoError(t, a.StartPingMonitor("s1", "10.0.0.1"))
	require.NoError(t, a.StartPingMonitor("s2", "10.0.0.2"))
	named := func(nm string) int {
		c := 0
		for _, e := range get() {
			if e.name == nm {
				c++
			}
		}
		return c
	}
	require.Eventually(t, func() bool { return named("metrics:s1") >= 2 && named("metrics:s2") >= 2 }, 2*time.Second, 5*time.Millisecond)
	a.StopPingMonitor("s1")
	s2Before := named("metrics:s2")
	require.Eventually(t, func() bool { return named("metrics:s2") > s2Before }, 2*time.Second, 5*time.Millisecond)
	a.StopPingMonitor("s2")
}
