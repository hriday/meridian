// Package app holds the Wails controller that binds Meridian's engine
// packages to the frontend. It contains no engine logic itself — each
// bound method delegates to a pure package under internal/.
package app

import "context"

// App is the Wails-bound controller. Its public methods are exposed to the
// frontend; unexported fields and methods are not, which is why the context
// accessor below is kept out of the public surface.
type App struct {
	ctx context.Context
}

// New returns a controller with no context; the context arrives at startup.
func New() *App {
	return &App{}
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
