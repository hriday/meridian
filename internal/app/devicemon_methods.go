package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/snmp"
)

// deviceInterval is the device-health poll cadence (slower than interface
// throughput; lighter on the gear). A seam for the streaming tests.
var deviceInterval = 5 * time.Second

// DeviceSnapshot is one poll of a device's health, emitted on "metrics:<id>".
// Thresholds live in the UI; the backend only reports values + reachability.
type DeviceSnapshot struct {
	Name   string  `json:"name"`
	CPUPct float64 `json:"cpuPct"`
	MemPct float64 `json:"memPct"`
	HasCPU bool    `json:"hasCpu"`
	HasMem bool    `json:"hasMem"`
	State  string  `json:"state"` // ok | unreachable | error
	Note   string  `json:"note"`
}

// StartDeviceMonitor streams a DeviceSnapshot per poll for one device's CPU and
// memory. System() is the reachability probe (it returns the transport error and
// the sysName); ResourceUsage() is best-effort and never errors, so a device
// that simply lacks the host-resources OIDs reports n/a, not an error.
func (a *App) StartDeviceMonitor(streamID, device, community string) error {
	device = strings.TrimSpace(device)
	if device == "" {
		return fmt.Errorf("enter a device address")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &monitor{cancel: cancel, done: make(chan struct{}), baseline: func() {}}
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
			emitEvent(emitCtx, event, DeviceSnapshot{State: "error",
				Note: "can't reach the device: " + scrub(err.Error(), community)})
			return
		}
		defer c.Close()

		first := true
		for {
			if ctx.Err() != nil {
				return
			}
			sys, err := snmp.System(c) // reachability probe
			if err != nil {
				if first {
					emitEvent(emitCtx, event, DeviceSnapshot{State: "error",
						Note: "SNMP read failed: " + scrub(err.Error(), community)})
					return
				}
				emitEvent(emitCtx, event, DeviceSnapshot{State: "unreachable",
					Note: "the device stopped responding — retrying"})
			} else {
				res, _ := snmp.ResourceUsage(c) // best-effort; n/a when absent
				emitEvent(emitCtx, event, DeviceSnapshot{
					Name:   sys.Name,
					CPUPct: res.CPUPct, MemPct: res.MemPct,
					HasCPU: res.HasCPU, HasMem: res.HasMem,
					State: "ok",
				})
			}
			first = false
			select {
			case <-ctx.Done():
				return
			case <-time.After(deviceInterval):
			}
		}
	}()
	return nil
}

// StopDeviceMonitor cancels a stream and blocks until its goroutine has exited.
func (a *App) StopDeviceMonitor(streamID string) { a.stopMonitor(streamID) }
