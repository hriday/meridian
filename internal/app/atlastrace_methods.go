package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hriday/meridian/internal/atlas"
	"github.com/hriday/meridian/internal/settings"
)

// atlasTrace is a seam over atlas.Trace so method tests stay offline.
var atlasTrace = atlas.Trace

const (
	atlasTraceDefaultProbes = 5
	atlasTraceMaxProbes     = 20
)

// AtlasTrace runs a one-off traceroute to target from worldwide RIPE Atlas
// probes using the locally stored key. probes is clamped to [1, 20] (default 5).
func (a *App) AtlasTrace(target string, probes int) (atlas.TraceReport, error) {
	key, err := settings.AtlasKey()
	if err != nil {
		return atlas.TraceReport{}, err
	}
	if key == "" {
		return atlas.TraceReport{}, fmt.Errorf("no RIPE Atlas key set — add your key in the Remote ping tool")
	}
	if probes < 1 {
		probes = atlasTraceDefaultProbes
	}
	if probes > atlasTraceMaxProbes {
		probes = atlasTraceMaxProbes
	}
	client := &http.Client{Timeout: atlasTimeout}
	ctx, cancel := context.WithTimeout(a.lookupCtx(), atlasTimeout)
	defer cancel()
	return atlasTrace(ctx, client, key, target, probes, atlasPollInterval, atlasTimeout)
}
