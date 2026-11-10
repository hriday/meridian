package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/atlas"
	"github.com/hriday/meridian/internal/settings"
)

// atlasPing is a seam over atlas.Ping so method tests stay offline.
var atlasPing = atlas.Ping

const (
	atlasDefaultProbes = 10
	atlasMaxProbes     = 50
	atlasPollInterval  = 5 * time.Second
	atlasTimeout       = 90 * time.Second
)

// AtlasKeyStatus reports whether a RIPE Atlas API key is stored and, if so, a
// masked hint of it. The full key is never returned to the frontend.
type AtlasKeyStatus struct {
	Set  bool   `json:"set"`
	Hint string `json:"hint"`
}

// AtlasKeyStatus returns whether an Atlas key is set plus a masked hint.
func (a *App) AtlasKeyStatus() (AtlasKeyStatus, error) {
	key, err := settings.AtlasKey()
	if err != nil {
		return AtlasKeyStatus{}, err
	}
	if key == "" {
		return AtlasKeyStatus{Set: false}, nil
	}
	return AtlasKeyStatus{Set: true, Hint: maskKey(key)}, nil
}

// maskKey returns a hint showing only the last 6 characters. For short keys it
// reveals nothing — the hint must never carry the full secret, whatever its
// length.
func maskKey(key string) string {
	if len(key) <= 6 {
		return "……"
	}
	return "…" + key[len(key)-6:]
}

// SetAtlasKey stores the operator's RIPE Atlas API key locally.
func (a *App) SetAtlasKey(key string) error {
	return settings.SetAtlasKey(key)
}

// ClearAtlasKey removes the stored RIPE Atlas API key.
func (a *App) ClearAtlasKey() error {
	return settings.ClearAtlasKey()
}

// AtlasPing runs a one-off ping to target from worldwide RIPE Atlas probes,
// using the locally stored key. probes is clamped to [1, 50] (default 10).
func (a *App) AtlasPing(target string, probes int) (atlas.Report, error) {
	key, err := settings.AtlasKey()
	if err != nil {
		return atlas.Report{}, err
	}
	if key == "" {
		return atlas.Report{}, fmt.Errorf("no RIPE Atlas key set — add your key in the Remote ping tool")
	}
	if probes < 1 {
		probes = atlasDefaultProbes
	}
	if probes > atlasMaxProbes {
		probes = atlasMaxProbes
	}
	client := &http.Client{Timeout: atlasTimeout}
	ctx, cancel := context.WithTimeout(a.lookupCtx(), atlasTimeout)
	defer cancel()
	return atlasPing(ctx, client, key, target, probes, atlasPollInterval, atlasTimeout)
}
