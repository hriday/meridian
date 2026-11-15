//go:build darwin

package main

import "github.com/wailsapp/wails/v2/pkg/menu"

// appMenu builds the native macOS menu bar: the application menu (About,
// Hide, Quit…), the Edit menu — which is what makes ⌘C / ⌘V / ⌘X / ⌘A / ⌘Z
// work inside the webview — and the Window menu (Minimize, Zoom, window list).
// Without a menu, macOS shows a bare default and the standard clipboard and
// undo shortcuts do nothing.
func appMenu() *menu.Menu {
	m := menu.NewMenu()
	m.Append(menu.AppMenu())    // on macOS this must be appended first
	m.Append(menu.EditMenu())   // clipboard + undo/redo shortcuts
	m.Append(menu.WindowMenu()) // Minimize / Zoom / Bring All to Front
	return m
}
