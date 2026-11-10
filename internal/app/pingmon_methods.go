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

type pingMonitor struct {
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
	agg    *metrics.Aggregator
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
	m := &pingMonitor{cancel: cancel, done: make(chan struct{}), agg: metrics.NewAggregator(pingWindow)}

	a.monMu.Lock()
	if a.monitors == nil {
		a.monitors = map[string]*pingMonitor{}
	}
	if _, exists := a.monitors[streamID]; exists {
		a.monMu.Unlock()
		cancel()
		return fmt.Errorf("a monitor with that id is already running")
	}
	a.monitors[streamID] = m
	a.monMu.Unlock()

	var pinger ping.Pinger = a.newPinger() // scan.Pinger satisfies ping.Pinger
	event := "metrics:" + streamID
	emitCtx := a.lookupCtx()

	go func() {
		defer close(m.done)
		available, err := ping.Monitor(ctx, pinger, target, pingInterval, pingTimeout, func(r ping.Reply) {
			m.mu.Lock()
			snap := m.agg.Add(r.RTTMillis, r.OK)
			m.mu.Unlock()
			emitEvent(emitCtx, event, snap)
		})
		if ctx.Err() != nil {
			return // stopped
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
func (a *App) StopPingMonitor(streamID string) {
	a.monMu.Lock()
	m := a.monitors[streamID]
	delete(a.monitors, streamID)
	a.monMu.Unlock()
	if m != nil {
		m.cancel()
		<-m.done
	}
}

// SetPingBaseline freezes the stream's current band as the known-good range.
func (a *App) SetPingBaseline(streamID string) {
	a.monMu.Lock()
	m := a.monitors[streamID]
	a.monMu.Unlock()
	if m != nil {
		m.mu.Lock()
		m.agg.SetBaseline()
		m.mu.Unlock()
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
		a.StopPingMonitor(id)
	}
}
