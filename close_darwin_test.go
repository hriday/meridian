//go:build darwin

package main

import (
	"testing"

	"github.com/hriday/meridian/internal/app"
)

// On macOS the close button must hide the window (app stays in the Dock), not
// quit — and the native menu must still be present so ⌘Q can quit.
func TestDarwinHidesWindowOnClose(t *testing.T) {
	if !hideWindowOnClose {
		t.Fatal("hideWindowOnClose is false on darwin — closing the window would quit the app")
	}
	opts := appOptions(app.New())
	if !opts.HideWindowOnClose {
		t.Error("appOptions.HideWindowOnClose is false on darwin")
	}
	if opts.Menu == nil {
		t.Error("appOptions.Menu is nil on darwin — ⌘Q / Quit would be missing")
	}
	if opts.Title != "Meridian" {
		t.Errorf("appOptions.Title = %q, want Meridian", opts.Title)
	}
}
