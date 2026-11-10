package app

import "github.com/hriday/meridian/internal/fwconfig"

// AnalyzeFirewall parses a firewall config export (FortiGate) offline and
// returns one analysis per VDOM: VLANs, "what talks to what" flows, counts.
// vendor defaults to "fortigate".
func (a *App) AnalyzeFirewall(configText, vendor string) ([]fwconfig.Analysis, error) {
	return fwconfig.Analyze(configText, vendor)
}
