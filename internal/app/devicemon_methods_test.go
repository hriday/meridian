package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/snmp"
	"github.com/stretchr/testify/require"
)

// fakeDev answers the System probe (sysDescr/sysName/sysUpTime) and the
// host-resources OIDs (hrProcessorLoad, hrStorage*), with switches to script
// transport failures.
type fakeDev struct {
	name            string
	cpu             int
	hasCPU          bool
	memSize, memUsed uint64
	hasMem          bool
	sysCalls        int64
	failSystem      bool
	failAfter       int // System fails after this many successful probes (0 = never)
}

func (f *fakeDev) Get(oids []string) ([]snmp.PDU, error) {
	if len(oids) > 0 && strings.HasPrefix(oids[0], "1.3.6.1.2.1.1.1") { // sysDescr → System probe
		n := atomic.AddInt64(&f.sysCalls, 1)
		if f.failSystem || (f.failAfter > 0 && int(n) > f.failAfter) {
			return nil, fmt.Errorf("read udp timeout community=%s", secret)
		}
		return []snmp.PDU{
			{OID: oids[0], Value: "a device"},
			{OID: oids[1], Value: f.name},
			{OID: oids[2], Value: uint(12345)},
		}, nil
	}
	out := make([]snmp.PDU, len(oids)) // storage Get
	for i, o := range oids {
		var v any = uint(0)
		switch {
		case strings.HasPrefix(o, "1.3.6.1.2.1.25.2.3.1.4"): // alloc units
			v = uint(1)
		case strings.HasPrefix(o, "1.3.6.1.2.1.25.2.3.1.5"): // size
			v = f.memSize
		case strings.HasPrefix(o, "1.3.6.1.2.1.25.2.3.1.6"): // used
			v = f.memUsed
		}
		out[i] = snmp.PDU{OID: o, Value: v}
	}
	return out, nil
}

func (f *fakeDev) Walk(root string) ([]snmp.PDU, error) {
	switch {
	case strings.HasPrefix(root, "1.3.6.1.2.1.25.3.3.1.2"): // hrProcessorLoad
		if !f.hasCPU {
			return nil, nil
		}
		return []snmp.PDU{{OID: root + ".1", Value: f.cpu}}, nil
	case strings.HasPrefix(root, "1.3.6.1.2.1.25.2.3.1.2"): // hrStorageType
		if !f.hasMem {
			return nil, nil
		}
		return []snmp.PDU{{OID: root + ".1", Value: "1.3.6.1.2.1.25.2.1.2"}}, nil // RAM type at idx 1
	}
	return nil, nil
}
func (f *fakeDev) Close() error { return nil }

// captureDevSnaps = captureSnaps (emitEvent + fast snmpInterval) plus a fast deviceInterval.
func captureDevSnaps(t *testing.T) func() []capSnap {
	get := captureSnaps(t)
	old := deviceInterval
	deviceInterval = time.Millisecond
	t.Cleanup(func() { deviceInterval = old })
	return get
}

func TestDeviceMonitorStreams(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, &fakeDev{name: "core-sw-1", cpu: 42, hasCPU: true, memSize: 100, memUsed: 55, hasMem: true}, nil)
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(DeviceSnapshot); ok && s.State == "ok" && s.CPUPct > 0 && s.Name == "core-sw-1" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	a.StopDeviceMonitor("s1")
	require.Equal(t, "metrics:s1", get()[0].name)
}

func TestDeviceMonitorScrubsCommunity(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, nil, fmt.Errorf("x509 bad host, community=%s", secret))
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(DeviceSnapshot); ok && s.State == "error" {
				require.NotContains(t, s.Note, secret)
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	a.StopDeviceMonitor("s1")
}

func TestDeviceMonitorSnapshotNoCommunity(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, &fakeDev{name: "n", cpu: 10, hasCPU: true, memSize: 10, memUsed: 3, hasMem: true}, nil)
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool { return len(get()) >= 3 }, 2*time.Second, 5*time.Millisecond)
	a.StopDeviceMonitor("s1")
	for _, e := range get() {
		b, _ := json.Marshal(e.data)
		require.NotContains(t, string(b), secret)
	}
}

func TestDeviceMonitorMissingResourcesNA(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, &fakeDev{name: "fw", hasCPU: false, hasMem: false}, nil) // reachable, no host-resources
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(DeviceSnapshot); ok && s.State == "ok" {
				require.False(t, s.HasCPU)
				require.False(t, s.HasMem)
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	a.StopDeviceMonitor("s1")
}

func TestDeviceMonitorFirstFailError(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, &fakeDev{failSystem: true}, nil)
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(DeviceSnapshot); ok && s.State == "error" {
				require.NotContains(t, s.Note, secret)
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	// goroutine should have exited after the single error snapshot
	n := len(get())
	time.Sleep(25 * time.Millisecond)
	a.StopDeviceMonitor("s1")
	require.Equal(t, n, len(get()))
}

func TestDeviceMonitorLaterUnreachable(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, &fakeDev{name: "n", cpu: 5, hasCPU: true, failAfter: 1}, nil) // ok once, then fails
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool {
		for _, e := range get() {
			if s, ok := e.data.(DeviceSnapshot); ok && s.State == "unreachable" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
	// stream keeps going after an unreachable tick
	n := len(get())
	require.Eventually(t, func() bool { return len(get()) > n }, 2*time.Second, 5*time.Millisecond)
	a.StopDeviceMonitor("s1")
}

func TestDeviceMonitorStopNoLeak(t *testing.T) {
	a := New()
	get := captureDevSnaps(t)
	setDial(t, &fakeDev{name: "n", cpu: 5, hasCPU: true}, nil)
	require.NoError(t, a.StartDeviceMonitor("s1", "10.0.0.1", secret))
	require.Eventually(t, func() bool { return len(get()) >= 2 }, 2*time.Second, 5*time.Millisecond)
	a.StopDeviceMonitor("s1")
	before := len(get())
	time.Sleep(25 * time.Millisecond)
	require.Equal(t, before, len(get()))
}
