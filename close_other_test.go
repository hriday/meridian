//go:build !darwin

package main

import (
	"testing"

	"github.com/hriday/meridian/internal/app"
)

// On Windows and Linux, closing the window should quit the app (the platform
// convention), so hide-on-close stays off.
func TestNonDarwinQuitsOnClose(t *testing.T) {
	if hideWindowOnClose {
		t.Fatal("hideWindowOnClose is true off darwin — close would hide instead of quit")
	}
	if appOptions(app.New()).HideWindowOnClose {
		t.Error("appOptions.HideWindowOnClose is true off darwin")
	}
}
