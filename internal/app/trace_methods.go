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
	gen := a.startTrace(cancel)
	defer a.endTrace(gen)

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

// startTrace registers cancel as the active trace (canceling any prior one) and
// returns a generation token.
func (a *App) startTrace(cancel context.CancelFunc) int {
	a.traceMu.Lock()
	defer a.traceMu.Unlock()
	if a.traceCancel != nil {
		a.traceCancel()
	}
	a.traceGen++
	a.traceCancel = cancel
	return a.traceGen
}

// endTrace clears the active cancel only if gen is still the current trace, so
// a finishing trace never clobbers a newer one that started meanwhile.
func (a *App) endTrace(gen int) {
	a.traceMu.Lock()
	defer a.traceMu.Unlock()
	if a.traceGen == gen {
		a.traceCancel = nil
	}
}
