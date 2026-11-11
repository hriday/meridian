package app

import "github.com/hriday/meridian/internal/fwconfig"

// AnalyzeFirewall parses a firewall config export offline and returns one
// analysis per context: segments, "what talks to what" flows, and counts.
// vendor is one of "fortigate", "panos", "asa", or "srx" (defaults to
// "fortigate").
func (a *App) AnalyzeFirewall(configText, vendor string) ([]fwconfig.Analysis, error) {
	return fwconfig.Analyze(configText, vendor)
}
