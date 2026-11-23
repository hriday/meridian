//go:build darwin

package main

// hideWindowOnClose: on macOS the close button hides the window (app stays in
// the Dock, ⌘Q quits), matching how native Mac apps behave.
const hideWindowOnClose = true
