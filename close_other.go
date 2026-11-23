//go:build !darwin

package main

// hideWindowOnClose: on Windows and Linux, closing the window quits the app —
// the expected behaviour on those platforms.
const hideWindowOnClose = false
