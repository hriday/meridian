package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/ble"
)

// BLESurvey scans for nearby Bluetooth Low Energy devices for the given number
// of seconds (0 = a sensible default) and returns them with signal, name and
// advertised company. Read-only; it only listens for advertisements. On macOS
// the first run prompts for Bluetooth access. User-initiated.
func (a *App) BLESurvey(seconds int) (ble.Survey, error) {
	dur := time.Duration(seconds) * time.Second
	switch {
	case dur <= 0:
		dur = 6 * time.Second
	case dur > 30*time.Second:
		dur = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), dur+10*time.Second)
	defer cancel()
	return ble.Scan(ctx, dur)
}
