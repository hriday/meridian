package app

import (
	"context"

	"github.com/hriday/meridian/internal/update"
	"github.com/hriday/meridian/internal/version"
)

// AppVersion returns the running build's version (e.g. "v0.1.0", or "dev").
func (a *App) AppVersion() string {
	return version.Version
}

// CheckForUpdate asks GitHub whether a newer Meridian release exists. It runs
// only when the user asks — there is no automatic background check — makes a
// single request to the public GitHub Releases API, and sends no identifying
// information.
func (a *App) CheckForUpdate() (update.Result, error) {
	return update.CheckLatest(context.Background(), version.Version)
}
