package app

import (
	"github.com/hriday/engineers_tools/internal/netinput"
	"github.com/hriday/engineers_tools/internal/subnet"
)

// These methods are bound to the frontend by Wails. Each is a thin
// delegation to a pure engine package — no logic lives here.

// Classify recognizes a free-text value (IP, prefix, ASN, domain, host).
func (a *App) Classify(input string) (netinput.Result, error) {
	return netinput.Classify(input)
}

// SubnetDescribe returns the full description of a CIDR block.
func (a *App) SubnetDescribe(cidr string) (subnet.Info, error) {
	return subnet.Describe(cidr)
}

// SubnetSplit divides a CIDR into equal subnets of the given prefix length.
func (a *App) SubnetSplit(cidr string, newPrefixLen int) ([]string, error) {
	return subnet.Split(cidr, newPrefixLen)
}
