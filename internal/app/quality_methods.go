package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hriday/meridian/internal/metrics"
	"github.com/hriday/meridian/internal/netquality"
	"github.com/hriday/meridian/internal/ping"
)

// QualitySnapshot is one streamed connection-quality reading: the ping metrics
// plus the derived MOS score.
type QualitySnapshot struct {
	metrics.Snapshot
	netquality.Score
}

// StartQualityMonitor begins a live connection-quality measurement to target,
// streaming a QualitySnapshot per probe on "quality:<streamID>" — latency,
// jitter, loss, and a rolling MOS verdict.
func (a *App) StartQualityMonitor(streamID, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("enter a host to measure")
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

	pinger := a.newPinger()
	event := "quality:" + streamID
	emitCtx := a.lookupCtx()

	go func() {
		defer close(m.done)
		available, err := ping.Monitor(ctx, pinger, target, pingInterval, pingTimeout, func(r ping.Reply) {
			mu.Lock()
			snap := agg.Add(r.RTTMillis, r.OK)
			mu.Unlock()
			score := netquality.Rate(snap.AvgMs, snap.JitterMs, snap.LossPct)
			emitEvent(emitCtx, event, QualitySnapshot{Snapshot: snap, Score: score})
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			emitEvent(emitCtx, event, QualitySnapshot{Snapshot: metrics.Snapshot{State: "error", Note: err.Error()}})
			return
		}
		if !available {
			emitEvent(emitCtx, event, QualitySnapshot{Snapshot: metrics.Snapshot{State: "error",
				Note: "ICMP isn't available — raw sockets may need elevated privileges on this OS."}})
		}
	}()
	return nil
}

// StopQualityMonitor ends a measurement and blocks until its goroutine exits.
func (a *App) StopQualityMonitor(streamID string) { a.stopMonitor(streamID) }
