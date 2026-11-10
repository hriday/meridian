package app

import (
	"errors"
	"time"

	"github.com/hriday/meridian/internal/snmp"
)

// SnmpSweep sweeps a CIDR for SNMP responders, streaming each probed host as a
// "snmp:host:<streamID>" event and returning the responders sorted by IP. The
// per-sweep event name keeps a prior sweep's late events from reaching a new
// sweep's progress counter. It runs under the shared single-scan context, so
// ScanStop cancels it.
func (a *App) SnmpSweep(streamID, cidr, community string) ([]snmp.SweepHost, error) {
	ctx := a.scanContext()
	event := "snmp:host:" + streamID
	res, err := snmp.Sweep(ctx, cidr, community, time.Second, 64,
		func(h snmp.SweepHost) { a.scanEmit(event, h) })
	if err != nil {
		return nil, errors.New(scrub(err.Error(), community))
	}
	return res, nil
}
