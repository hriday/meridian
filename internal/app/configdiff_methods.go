package app

import "github.com/hriday/meridian/internal/configdiff"

// DiffConfigs compares two device configurations line by line, attributing each
// change to its config section. Pure and offline.
func (a *App) DiffConfigs(oldText, newText string) configdiff.Result {
	return configdiff.Diff(oldText, newText)
}
