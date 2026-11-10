package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/metrics"
	"github.com/hriday/meridian/internal/snmp"
	"github.com/stretchr/testify/require"
)

const secret = "SEKRETcommunity"

type fakeSNMP struct {
	in      int64
	failGet bool
}

func (f *fakeSNMP) Get(oids []string) ([]snmp.PDU, error) {
	if f.failGet {
		return nil, fmt.Errorf("read udp timeout community=%s", secret)
	}
	io := uint64(atomic.AddInt64(&f.in, 1_000_000))
	out := make([]snmp.PDU, len(oids))
	for i, o := range oids {
		var v any = uint(0)
		switch {
		case strings.HasPrefix(o, "1.3.6.1.2.1.31.1.1.1.6"): // ifHCInOctets
			v = io
		case strings.HasPrefix(o, "1.3.6.1.2.1.31.1.1.1.10"): // ifHCOutOctets
			v = io / 2
		case strings.HasPrefix(o, "1.3.6.1.2.1.2.2.1.8"): // ifOperStatus
			v = 1
		}
		out[i] = snmp.PDU{OID: o, Value: v}
	}
	return out, nil
}
func (f *fakeSNMP) Walk(string) ([]snmp.PDU, error) { return nil, nil }
func (f *fakeSNMP) Close() error                    { return nil }

type capSnap struct {
	name string
	data any
}

func captureSnaps(t *testing.T) func() []capSnap {
	t.Helper()
	var mu sync.Mutex
	var got []capSnap
	oldEmit := emitEvent
	emitEvent = func(_ context.Context, name string, data ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if len(data) == 1 {
			got = append(got, capSnap{name, data[0]})
		}
	}
	oldInt := snmpInterval
	snmpInterval = time.Millisecond
	t.Cleanup(func() { emitEvent = oldEmit; snmpInterval = oldInt })
	return func() []capSnap { mu.Lock(); defer mu.Unlock(); return append([]capSnap(nil), got...) }
}

func setDial(t *testing.T, c snmp.Client, err error) {
	t.Helper()
	old := dial
	dial = func(string, string, time.Duration) (snmp.Client, error) { return c, err }
	t.Cleanup(func() { dial = old })
}

func TestListInterfacesScrubsCommunity(t *testing.T) {
	setDial(t, nil, fmt.Errorf("x509 bad host, community=%s", secret))
	_, err := New().ListInterfaces("10.0.0.1", secret)
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
}

func TestIfMonitorStreams(t *testing.T) {
	a := New()
	get := captureSnaps(t)
	setDial(t, &fakeSNMP{}, nil)
	require.NoError(t, a.StartIfMonitor("s1", "10.0.0.1", secret, 1, 1_000_000_000))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(metrics.ThruSnapshot); ok && s.InBps > 0 {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	a.StopIfMonitor("s1")
	require.Equal(t, "metrics:s1", get()[0].name)
}

func TestIfMonitorSnapshotNoCommunity(t *testing.T) {
	a := New()
	get := captureSnaps(t)
	setDial(t, &fakeSNMP{}, nil)
	require.NoError(t, a.StartIfMonitor("s1", "10.0.0.1", secret, 1, 1_000_000_000))
	require.Eventually(t, func() bool { return len(get()) >= 3 }, 2*time.Second, 5*time.Millisecond)
	a.StopIfMonitor("s1")
	for _, e := range get() {
		b, _ := json.Marshal(e.data)
		require.NotContains(t, string(b), secret)
	}
}

func TestIfMonitorStopNoLeak(t *testing.T) {
	a := New()
	get := captureSnaps(t)
	setDial(t, &fakeSNMP{}, nil)
	require.NoError(t, a.StartIfMonitor("s1", "10.0.0.1", secret, 1, 1_000_000_000))
	require.Eventually(t, func() bool { return len(get()) >= 2 }, 2*time.Second, 5*time.Millisecond)
	a.StopIfMonitor("s1")
	before := len(get())
	time.Sleep(25 * time.Millisecond)
	require.Equal(t, before, len(get()))
}

func TestIfMonitorUnreachable(t *testing.T) {
	a := New()
	get := captureSnaps(t)
	setDial(t, &fakeSNMP{failGet: true}, nil)
	require.NoError(t, a.StartIfMonitor("s1", "10.0.0.1", secret, 1, 1_000_000_000))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(metrics.ThruSnapshot); ok && s.State == "error" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	a.StopIfMonitor("s1")
}

func TestSetIfBaseline(t *testing.T) {
	a := New()
	get := captureSnaps(t)
	setDial(t, &fakeSNMP{}, nil)
	require.NoError(t, a.StartIfMonitor("s1", "10.0.0.1", secret, 1, 1_000_000_000))
	require.Eventually(t, func() bool { return len(get()) >= 6 }, 2*time.Second, 5*time.Millisecond)
	a.SetIfBaseline("s1") // must not panic; stream continues
	n := len(get())
	require.Eventually(t, func() bool { return len(get()) > n }, 2*time.Second, 5*time.Millisecond)
	a.StopIfMonitor("s1")
}
