package app

import (
	"context"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/trace"
	"github.com/hriday/meridian/internal/tracerun"
)

// Traceroute traces the path to target (IPv4 unless useV6), emitting a
// "trace:hop" event per hop as it streams and returning the final Result.
func (a *App) Traceroute(target string, useV6 bool) (trace.Result, error) {
	ctx, cancel := context.WithCancel(a.lookupCtx())
	a.setTraceCancel(cancel)
	defer a.setTraceCancel(nil)

	return tracerun.Run(ctx, target, useV6, func(h trace.Hop) {
		if a.ctx != nil {
			wruntime.EventsEmit(a.ctx, "trace:hop", h)
		}
	})
}

// StopTrace cancels the trace currently in flight, if any.
func (a *App) StopTrace() {
	a.traceMu.Lock()
	cancel := a.traceCancel
	a.traceMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *App) setTraceCancel(c context.CancelFunc) {
	a.traceMu.Lock()
	// cancel any previous in-flight trace when starting a new one
	if c != nil && a.traceCancel != nil {
		a.traceCancel()
	}
	a.traceCancel = c
	a.traceMu.Unlock()
}
