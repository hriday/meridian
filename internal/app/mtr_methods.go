package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/mtr"
	"github.com/hriday/meridian/internal/tracerun"
)

// mtrInterval is the pause between route sweeps. Seam for tests.
var mtrInterval = time.Second

// StartMTR begins a continuous mtr-style path monitor to target, streaming an
// mtr.Update snapshot per round on the Wails event "mtr:<streamID>". The
// frontend generates the id and subscribes before calling, so no round is
// missed.
func (a *App) StartMTR(streamID, target string, useV6 bool) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("enter a host to trace")
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &monitor{cancel: cancel, done: make(chan struct{})}
	if err := a.registerMonitor(streamID, m); err != nil {
		cancel()
		return err
	}

	event := "mtr:" + streamID
	emitCtx := a.lookupCtx()

	go func() {
		defer close(m.done)
		err := mtr.Monitor(ctx, tracerun.Run, target, useV6, mtrInterval, func(u mtr.Update) {
			emitEvent(emitCtx, event, u)
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			emitEvent(emitCtx, event, mtr.Update{Target: target, Hops: nil,
				Addr: "error: " + err.Error()})
		}
	}()
	return nil
}

// StopMTR cancels a path monitor and blocks until its goroutine has exited.
func (a *App) StopMTR(streamID string) { a.stopMonitor(streamID) }
