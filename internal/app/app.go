// Package app holds the Wails controller that binds Meridian's engine
// packages to the frontend. It contains no engine logic itself — each
// bound method delegates to a pure package under internal/.
package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/lookup"
)

// App is the Wails-bound controller. Its public methods are exposed to the
// frontend; unexported fields and methods are not, which is why the context
// accessor below is kept out of the public surface.
type App struct {
	ctx    context.Context
	lookup *lookup.Client
}

// New returns a controller with no context; the context arrives at startup.
func New() *App {
	return &App{
		lookup: lookup.NewClient(&http.Client{Timeout: 5 * time.Second}, 10*time.Minute),
	}
}

// OnStartup returns the Wails OnStartup callback for a. startup stays
// unexported so Wails does not bind it (or a context accessor) as a
// frontend-callable method.
func OnStartup(a *App) func(context.Context) {
	return a.startup
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// lookupCtx is the context for outbound lookups: the Wails startup context
// once set, else a background context (e.g. in tests).
func (a *App) lookupCtx() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}
