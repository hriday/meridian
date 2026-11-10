package app

import (
	"errors"
	"time"

	"github.com/hriday/meridian/internal/snmp"
)

// SnmpSweep sweeps a CIDR for SNMP responders, streaming each probed host as a
// "snmp:host" event and returning the responders sorted by IP. It runs under the
// shared single-scan context, so ScanStop cancels it.
func (a *App) SnmpSweep(cidr, community string) ([]snmp.SweepHost, error) {
	ctx := a.scanContext()
	res, err := snmp.Sweep(ctx, cidr, community, time.Second, 64,
		func(h snmp.SweepHost) { a.scanEmit("snmp:host", h) })
	if err != nil {
		return nil, errors.New(scrub(err.Error(), community))
	}
	return res, nil
}
