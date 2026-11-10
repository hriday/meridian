package app

import (
	"context"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/scan"
)

func (a *App) scanEmit(event string, data any) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, event, data)
	}
}

// scanContext returns a cancellable context for a scan, cancelling any previous one.
func (a *App) scanContext() context.Context {
	a.scanMu.Lock()
	defer a.scanMu.Unlock()
	if a.scanCancel != nil {
		a.scanCancel()
	}
	ctx, cancel := context.WithCancel(a.lookupCtx())
	a.scanCancel = cancel
	return ctx
}

// ScanStop cancels the running scan.
func (a *App) ScanStop() {
	a.scanMu.Lock()
	defer a.scanMu.Unlock()
	if a.scanCancel != nil {
		a.scanCancel()
		a.scanCancel = nil
	}
}

// ScanCommonPorts is the preset of common ports.
func (a *App) ScanCommonPorts() []int { return scan.CommonPorts() }

// PortScan TCP-connect-scans host's ports, streaming `scan:port` events.
func (a *App) PortScan(host string, ports []int) ([]scan.PortResult, error) {
	ctx := a.scanContext()
	return scan.ScanPorts(ctx, host, ports, 2*time.Second, 128,
		func(r scan.PortResult) { a.scanEmit("scan:port", r) })
}

// PingSweep sweeps a CIDR for live hosts, streaming `scan:host` events.
func (a *App) PingSweep(cidr string, tcpPorts []int) ([]scan.HostResult, error) {
	ctx := a.scanContext()
	return scan.Sweep(ctx, cidr, a.newPinger(), tcpPorts, time.Second, 64,
		func(r scan.HostResult) { a.scanEmit("scan:host", r) })
}
