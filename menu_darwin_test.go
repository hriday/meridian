//go:build darwin

package main

import "testing"

// appMenu must compose the three native menus so macOS gets the application
// menu (About/Quit), the Edit menu (clipboard shortcuts) and the Window menu.
func TestAppMenuHasAppEditWindow(t *testing.T) {
	m := appMenu()
	if m == nil {
		t.Fatal("appMenu() is nil on darwin — no menu bar, no clipboard shortcuts")
	}
	if got := len(m.Items); got != 3 {
		t.Fatalf("menu has %d top-level items, want 3 (App, Edit, Window)", got)
	}
}
