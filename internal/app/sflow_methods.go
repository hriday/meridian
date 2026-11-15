package app

import (
	"net/netip"
	"time"

	"github.com/hriday/meridian/internal/netflow"
)

// sFlow decodes into the same FlowRecord model as NetFlow, so its aggregated
// top-talkers snapshot reuses NetflowView.

// SflowStart binds the sFlow receiver to a UDP port (0 → OS-chosen); start
// while running restarts it. A bind failure is returned.
func (a *App) SflowStart(port int) error { return a.sflow.Start(port) }

// SflowStop stops the receiver and discards its buffer.
func (a *App) SflowStop() { a.sflow.Stop() }

// SflowTop aggregates the sampled flows received within the window into the top
// talkers (by bytes, or packets). Bytes/packets are sampling-rate-scaled
// estimates, since sFlow carries 1-in-N sampled headers.
func (a *App) SflowTop(windowSec int, byPackets bool) NetflowView {
	running, port := a.sflow.Status()
	if windowSec <= 0 {
		windowSec = 60
	}
	now := time.Now()
	window := time.Duration(windowSec) * time.Second
	cutoff := now.Add(-window)
	recs := a.sflow.Snapshot(func(f netflow.FlowRecord) bool { return !f.T.Before(cutoff) })

	exporters := map[netip.Addr]struct{}{}
	for _, f := range recs {
		exporters[f.Exporter] = struct{}{}
	}
	top, covered := netflow.TopTalkers(recs, window, now, byPackets, 50)
	truncated := a.sflow.Count() >= netflowBuffer && covered < windowSec-2
	return NetflowView{
		Running: running, Port: port,
		Flows: len(recs), Exporters: len(exporters), WindowSecCovered: covered,
		Truncated: truncated,
		Top:       top,
	}
}
