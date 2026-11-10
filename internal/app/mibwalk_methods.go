package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hriday/meridian/internal/snmp"
)

// mibCap bounds a walk so an over-broad subtree can't flood the UI.
const mibCap = 5000

// MIBWalk walks the OID subtree of a device and returns each OID with its type
// and value (read-only). The community is scrubbed from any error.
func (a *App) MIBWalk(device, community, oid string) ([]snmp.MIBRow, error) {
	device = strings.TrimSpace(device)
	if device == "" {
		return nil, fmt.Errorf("enter a device address")
	}
	oid = strings.TrimPrefix(strings.TrimSpace(oid), ".")
	if oid == "" {
		return nil, fmt.Errorf("enter an OID")
	}
	c, err := dial(device, community, snmpTimeout)
	if err != nil {
		return nil, errors.New(scrub(err.Error(), community))
	}
	defer c.Close()
	rows, err := snmp.MIBWalk(c, oid, mibCap)
	if err != nil {
		return nil, errors.New(scrub(err.Error(), community))
	}
	return rows, nil
}
