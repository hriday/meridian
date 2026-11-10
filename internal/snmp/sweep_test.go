package snmp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// sweepFake implements Client: when ok it answers the System probe, else its Get
// returns a transport error (a non-responding host).
type sweepFake struct {
	name, descr string
	uptime      uint
	ok          bool
}

func (f sweepFake) Get(oids []string) ([]PDU, error) {
	if !f.ok {
		return nil, fmt.Errorf("read udp timeout")
	}
	out := make([]PDU, len(oids))
	for i, o := range oids {
		switch {
		case strings.HasPrefix(o, oidSysDescr):
			out[i] = PDU{OID: o, Value: f.descr}
		case strings.HasPrefix(o, oidSysName):
			out[i] = PDU{OID: o, Value: f.name}
		case strings.HasPrefix(o, oidSysUpTime):
			out[i] = PDU{OID: o, Value: f.uptime}
		default:
			out[i] = PDU{OID: o}
		}
	}
	return out, nil
}
func (sweepFake) Walk(string) ([]PDU, error) { return nil, nil }
func (sweepFake) Close() error               { return nil }

func setSweepDial(t *testing.T, fn func(string, string, time.Duration) (Client, error)) {
	t.Helper()
	old := sweepDial
	sweepDial = fn
	t.Cleanup(func() { sweepDial = old })
}

const community = "public"

func TestSweepFindsResponders(t *testing.T) {
	setSweepDial(t, func(device, _ string, _ time.Duration) (Client, error) {
		switch device {
		case "10.0.0.1":
			return sweepFake{name: "core-sw-1", descr: "Cisco IOS", uptime: 1000, ok: true}, nil
		case "10.0.0.3":
			return sweepFake{name: "edge-rtr-1", descr: "IOS XE", uptime: 2000, ok: true}, nil
		default:
			return sweepFake{ok: false}, nil
		}
	})
	var scanned int32
	res, err := Sweep(context.Background(), "10.0.0.0/29", community, time.Second, 8,
		func(SweepHost) { atomic.AddInt32(&scanned, 1) })
	require.NoError(t, err)
	require.Len(t, res, 2) // only responders returned
	require.Equal(t, "10.0.0.1", res[0].IP)
	require.Equal(t, "core-sw-1", res[0].Name)
	require.Equal(t, "Cisco IOS", res[0].Descr)
	require.EqualValues(t, 1000, res[0].UptimeTicks)
	require.Equal(t, "10.0.0.3", res[1].IP) // sorted by IP
	require.EqualValues(t, 6, atomic.LoadInt32(&scanned)) // /29 minus network+broadcast
}

func TestSweepRejectsHugeCIDR(t *testing.T) {
	var dialed int32
	setSweepDial(t, func(string, string, time.Duration) (Client, error) {
		atomic.AddInt32(&dialed, 1)
		return sweepFake{ok: false}, nil
	})
	_, err := Sweep(context.Background(), "10.0.0.0/8", community, time.Second, 8, nil)
	require.Error(t, err)
	require.EqualValues(t, 0, atomic.LoadInt32(&dialed)) // rejected before any dial
}

func TestSweepInvalidCIDR(t *testing.T) {
	_, err := Sweep(context.Background(), "not-a-cidr", community, time.Second, 8, nil)
	require.Error(t, err)
}

func TestSweepNoCommunityInResult(t *testing.T) {
	setSweepDial(t, func(string, string, time.Duration) (Client, error) {
		return sweepFake{name: "n", descr: "d", uptime: 1, ok: true}, nil
	})
	res, err := Sweep(context.Background(), "10.0.0.0/30", community, time.Second, 8, nil)
	require.NoError(t, err)
	b, _ := json.Marshal(res)
	require.NotContains(t, string(b), community)
}

func TestSweepCancel(t *testing.T) {
	setSweepDial(t, func(string, string, time.Duration) (Client, error) {
		return sweepFake{ok: false}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	res, err := Sweep(ctx, "10.0.0.0/24", community, time.Second, 8, nil)
	require.NoError(t, err)
	require.Empty(t, res) // nothing responded; returns promptly without hanging
}
