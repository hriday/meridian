package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/wifi"
)

// WifiSupported reports whether Wi-Fi analysis works on this OS (macOS for now).
func (a *App) WifiSupported() bool { return wifi.Supported() }

// WifiStatus returns the Wi-Fi/RF snapshot — the connected network's channel,
// band, signal/SNR, and the neighboring APs with co-channel flags.
func (a *App) WifiStatus() (wifi.Info, error) {
	ctx, cancel := context.WithTimeout(a.lookupCtx(), 15*time.Second)
	defer cancel()
	return wifi.Status(ctx)
}

// WifiRequestLocation asks macOS for Location Services authorization so the
// Wi-Fi analyzer can show network names (SSIDs), which macOS otherwise hides.
// The system prompt appears the first time; after that it's resolved in System
// Settings. Returns immediately — the caller re-scans to pick up the result.
func (a *App) WifiRequestLocation() { wifi.RequestLocation() }
