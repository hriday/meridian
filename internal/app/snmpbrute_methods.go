package app

import (
	"context"
	"fmt"
	"time"

	"github.com/hriday/meridian/internal/snmp"
)

// SnmpBruteCommunities tries a list of SNMP community strings against a device
// and returns those that grant read access. It is an authorization-gated audit
// tool for your own gear: the caller must pass authorized=true (the UI requires
// an explicit confirmation). It reads only (sysDescr) and never writes; the
// community strings are used only to connect and are never logged.
func (a *App) SnmpBruteCommunities(device string, communities []string, authorized bool) ([]string, error) {
	if !authorized {
		return nil, fmt.Errorf("confirm you are authorized to test this device before running")
	}
	return snmp.BruteCommunities(context.Background(), device, communities, time.Second, 10)
}
