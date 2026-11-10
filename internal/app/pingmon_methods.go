package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/metrics"
	"github.com/hriday/meridian/internal/ping"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Seams so the streaming tests stay offline and fast.
var (
	emitEvent = func(ctx context.Context, name string, data ...interface{}) {
		wruntime.EventsEmit(ctx, name, data...)
	}
	pingInterval = time.Second
	pingTimeout  = 900 * time.Millisecond
)

const pingWindow = 120

// monitor is a running realtime stream (ping or SNMP): its cancel, a done
// channel that closes when the goroutine exits, and a baseline hook.
type monitor struct {
	cancel   context.CancelFunc
	done     chan struct{}
	baseline func()
}

func (a *App) registerMonitor(id string, m *monitor) error {
	a.monMu.Lock()
	defer a.monMu.Unlock()
	if a.monitors == nil {
		a.monitors = map[string]*monitor{}
	}
	if _, exists := a.monitors[id]; exists {
		return fmt.Errorf("a monitor with that id is already running")
	}
	a.monitors[id] = m
	return nil
}

// stopMonitor cancels a stream and blocks until its goroutine has exited.
func (a *App) stopMonitor(id string) {
	a.monMu.Lock()
	m := a.monitors[id]
	delete(a.monitors, id)
	a.monMu.Unlock()
	if m != nil {
		m.cancel()
		<-m.done
	}
}

func (a *App) monitorBaseline(id string) {
	a.monMu.Lock()
	m := a.monitors[id]
	a.monMu.Unlock()
	if m != nil && m.baseline != nil {
		m.baseline()
	}
}

func (a *App) stopAllMonitors() {
	a.monMu.Lock()
	ids := make([]string, 0, len(a.monitors))
	for id := range a.monitors {
		ids = append(ids, id)
	}
	a.monMu.Unlock()
	for _, id := range ids {
		a.stopMonitor(id)
	}
}

// StartPingMonitor begins a live ping to target, streaming a metrics.Snapshot
// per probe on the Wails event "metrics:<streamID>". The frontend generates the
// id and subscribes before calling this, so no snapshot is missed.
func (a *App) StartPingMonitor(streamID, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("enter a host to ping")
	}
	ctx, cancel := context.WithCancel(context.Background())
	agg := metrics.NewAggregator(pingWindow)
	var mu sync.Mutex
	m := &monitor{cancel: cancel, done: make(chan struct{}),
		baseline: func() { mu.Lock(); agg.SetBaseline(); mu.Unlock() }}
	if err := a.registerMonitor(streamID, m); err != nil {
		cancel()
		return err
	}

	var pinger ping.Pinger = a.newPinger()
	event := "metrics:" + streamID
	emitCtx := a.lookupCtx()

	go func() {
		defer close(m.done)
		available, err := ping.Monitor(ctx, pinger, target, pingInterval, pingTimeout, func(r ping.Reply) {
			mu.Lock()
			snap := agg.Add(r.RTTMillis, r.OK)
			mu.Unlock()
			emitEvent(emitCtx, event, snap)
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			emitEvent(emitCtx, event, metrics.Snapshot{State: "error", Note: err.Error()})
			return
		}
		if !available {
			emitEvent(emitCtx, event, metrics.Snapshot{State: "error",
				Note: "ICMP isn't available — raw sockets may need elevated privileges on this OS."})
		}
	}()
	return nil
}

// StopPingMonitor cancels a stream and blocks until its goroutine has exited.
func (a *App) StopPingMonitor(streamID string) { a.stopMonitor(streamID) }

// SetPingBaseline freezes the stream's current band as the known-good range.
func (a *App) SetPingBaseline(streamID string) { a.monitorBaseline(streamID) }
