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
