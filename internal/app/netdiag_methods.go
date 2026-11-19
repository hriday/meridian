package app

import (
	"context"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/netdiag"
)

// netdiagRun is the engine seam; tests override it to avoid real probes.
var netdiagRun = netdiag.Run

// NetworkDiagnostic runs the Local Network Diagnostic against target and
// returns the full report (all steps) synchronously.
func (a *App) NetworkDiagnostic(target string) (netdiag.Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return netdiagRun(ctx, netdiag.Options{Target: strings.TrimSpace(target)}, nil)
}
