package app

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/peripherals"
)

// PeripheralsSurvey inventories the USB devices attached to this machine —
// mice, keyboards, wireless receivers, storage, serial adapters, cameras — with
// their vendor, kind and any security call-outs. Read-only and instantaneous:
// it asks the OS what is attached, it does not open or probe any device.
// User-initiated.
func (a *App) PeripheralsSurvey() (peripherals.Survey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return peripherals.Scan(ctx)
}
