package app

import (
	"net/netip"
	"time"

	"github.com/hriday/meridian/internal/netflow"
)

// NetflowView is the aggregated top-talkers snapshot the frontend polls.
type NetflowView struct {
	Running          bool                   `json:"running"`
	Port             int                    `json:"port"`
	Flows            int                    `json:"flows"`
	Exporters        int                    `json:"exporters"`
	WindowSecCovered int                    `json:"windowSecCovered"`
	Truncated        bool                   `json:"truncated"` // ring saturated → older in-window flows evicted
	Top              []netflow.Conversation `json:"top"`
}

// NetflowStart binds the NetFlow receiver to a UDP port (0 → OS-chosen); start
// while running restarts it. A bind failure is returned.
func (a *App) NetflowStart(port int) error { return a.netflow.Start(port) }

// NetflowStop stops the receiver and discards its buffer.
func (a *App) NetflowStop() { a.netflow.Stop() }

// NetflowTop aggregates the flows received within the window into the top talkers
// (by bytes, or packets), plus in-window flow/exporter counts and how much of the
// window the retained records cover.
func (a *App) NetflowTop(windowSec int, byPackets bool) NetflowView {
	running, port := a.netflow.Status()
	if windowSec <= 0 {
		windowSec = 60
	}
	now := time.Now()
	window := time.Duration(windowSec) * time.Second
	cutoff := now.Add(-window)
	recs := a.netflow.Snapshot(func(f netflow.FlowRecord) bool { return !f.T.Before(cutoff) })

	exporters := map[netip.Addr]struct{}{}
	for _, f := range recs {
		exporters[f.Exporter] = struct{}{}
	}
	top, covered := netflow.TopTalkers(recs, window, now, byPackets, 50)
	// Truncation is real only when the ring is saturated (so older flows are
	// being evicted) AND the retained data doesn't reach back the full window —
	// not merely because the collector just started or traffic is sparse.
	truncated := a.netflow.Count() >= netflowBuffer && covered < windowSec-2
	return NetflowView{
		Running: running, Port: port,
		Flows: len(recs), Exporters: len(exporters), WindowSecCovered: covered,
		Truncated: truncated,
		Top:       top,
	}
}
