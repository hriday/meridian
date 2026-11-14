package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/svcstatus"
)

// ServiceStatus reports whether an online service is having an outage, from its
// public Statuspage.io feed. target is a known slug (e.g. "github") or a
// status-page URL. Free and keyless.
func (a *App) ServiceStatus(target string) (svcstatus.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return svcstatus.Check(ctx, &http.Client{Timeout: 12 * time.Second}, target)
}

// ServiceStatusKnown lists the built-in service slugs for the picker.
func (a *App) ServiceStatusKnown() []string {
	return svcstatus.KnownSlugs()
}
