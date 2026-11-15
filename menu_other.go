//go:build !darwin

package main

import "github.com/wailsapp/wails/v2/pkg/menu"

// appMenu returns no menu on Windows and Linux, which keeps the chrome-free
// look; the webview there provides the standard clipboard shortcuts natively,
// so no menu bar is needed.
func appMenu() *menu.Menu { return nil }
