package main

import (
	"embed"
	"fmt"

	"github.com/hriday/meridian/internal/app"
	"github.com/hriday/meridian/internal/version"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// appOptions builds the Wails configuration. It's separated from main so the
// platform-specific bits (menu, close behaviour) can be asserted in tests.
func appOptions(a *app.App) *options.App {
	return &options.App{
		Title:  "Meridian",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 17, B: 21, A: 1},
		Menu:             appMenu(),
		// On macOS the red close button should hide the window and leave the app
		// in the Dock (the platform convention), not quit — ⌘Q / Quit still exits.
		// Windows and Linux keep close-to-quit, which is what people expect there.
		HideWindowOnClose: hideWindowOnClose,
		OnStartup:         app.OnStartup(a),
		OnShutdown:        app.OnShutdown(a),
		Bind: []interface{}{
			a,
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title: "Meridian",
				Message: fmt.Sprintf(
					"Version %s\n\nA cross-platform desktop toolkit for understanding networks, addresses, and routes.\n\nApache-2.0 · © 2026 Hriday Balachandran",
					version.Version,
				),
				Icon: icon,
			},
		},
	}
}

func main() {
	a := app.New()
	if err := wails.Run(appOptions(a)); err != nil {
		println("Error:", err.Error())
	}
}
