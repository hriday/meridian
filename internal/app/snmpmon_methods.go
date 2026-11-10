package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/metrics"
	"github.com/hriday/meridian/internal/snmp"
)

// Seams for the streaming tests.
var (
	dial         = snmp.Dial
	snmpInterval = 2 * time.Second
	snmpTimeout  = 3 * time.Second
)

const snmpWindow = 120

// scrub removes the community string from a message so it never surfaces.
func scrub(msg, community string) string {
	if community == "" {
		return msg
	}
	return strings.ReplaceAll(msg, community, "***")
}

// ListInterfaces dials the device and returns its interface table for the picker.
func (a *App) ListInterfaces(device, community string) ([]snmp.Interface, error) {
	device = strings.TrimSpace(device)
	if device == "" {
		return nil, fmt.Errorf("enter a device address")
	}
	c, err := dial(device, community, snmpTimeout)
	if err != nil {
		return nil, errors.New(scrub(err.Error(), community))
	}
	defer c.Close()
	ifs, err := snmp.ListInterfaces(c)
	if err != nil {
		return nil, errors.New(scrub(err.Error(), community))
	}
	return ifs, nil
}

// StartIfMonitor streams a metrics.ThruSnapshot per poll on "metrics:<streamID>"
// for one interface of a device.
func (a *App) StartIfMonitor(streamID, device, community string, ifIndex int, speedBps uint64) error {
	device = strings.TrimSpace(device)
	if device == "" {
		return fmt.Errorf("enter a device address")
	}
	ctx, cancel := context.WithCancel(context.Background())
	agg := metrics.NewThruAgg(snmpWindow, speedBps)
	var mu sync.Mutex
	m := &monitor{cancel: cancel, done: make(chan struct{}),
		baseline: func() { mu.Lock(); agg.SetBaseline(); mu.Unlock() }}
	if err := a.registerMonitor(streamID, m); err != nil {
		cancel()
		return err
	}

	event := "metrics:" + streamID
	emitCtx := a.lookupCtx()

	go func() {
		defer close(m.done)
		c, err := dial(device, community, snmpTimeout)
		if err != nil {
			emitEvent(emitCtx, event, metrics.ThruSnapshot{State: "error",
				Note: "can't reach the device: " + scrub(err.Error(), community)})
			return
		}
		defer c.Close()

		first := true
		for {
			if ctx.Err() != nil {
				return
			}
			ctr, err := snmp.InterfaceCounters(c, ifIndex)
			if err != nil {
				if first {
					emitEvent(emitCtx, event, metrics.ThruSnapshot{State: "error",
						Note: "SNMP read failed: " + scrub(err.Error(), community)})
					return
				}
				emitEvent(emitCtx, event, metrics.ThruSnapshot{State: "unreachable",
					Note: "the device stopped responding — retrying"})
			} else {
				res, _ := snmp.ResourceUsage(c) // best-effort
				mu.Lock()
				snap := agg.Add(metrics.ThruSample{
					InOctets: ctr.InOctets, OutOctets: ctr.OutOctets,
					InErrors: ctr.InErrors, OutErrors: ctr.OutErrors,
					Bits: ctr.Bits, OperStatus: ctr.OperStatus,
					CPUPct: res.CPUPct, MemPct: res.MemPct, HasCPU: res.HasCPU, HasMem: res.HasMem,
					T: time.Now(),
				})
				mu.Unlock()
				emitEvent(emitCtx, event, snap)
			}
			first = false
			select {
			case <-ctx.Done():
				return
			case <-time.After(snmpInterval):
			}
		}
	}()
	return nil
}

// StopIfMonitor cancels a stream and blocks until its goroutine has exited.
func (a *App) StopIfMonitor(streamID string) { a.stopMonitor(streamID) }

// SetIfBaseline freezes the stream's current throughput band.
func (a *App) SetIfBaseline(streamID string) { a.monitorBaseline(streamID) }
