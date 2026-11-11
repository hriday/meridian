package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/snmp"
)

// SwitchPortMap walks a switch over SNMPv2c and returns the MAC-to-port table
// (joined with interface names and the ARP table). The community string is used
// only to open the session and never appears in an error or log.
func (a *App) SwitchPortMap(device, community string) ([]snmp.PortEntry, error) {
	device = strings.TrimSpace(device)
	if device == "" {
		return nil, fmt.Errorf("enter the switch's IP or hostname")
	}
	if strings.TrimSpace(community) == "" {
		community = "public"
	}
	c, err := snmp.Dial(device, community, 4*time.Second)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach %s over SNMP", device)
	}
	defer c.Close()
	return snmp.SwitchPorts(c)
}
