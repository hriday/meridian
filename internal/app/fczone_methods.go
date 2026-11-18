package app

import "github.com/hriday/meridian/internal/fczone"

// AnalyzeFCZoning parses Fibre Channel switch zoning (Cisco MDS `show zoneset`
// output or running-config, or Brocade `cfgshow`) and checks it for SAN hygiene
// problems. Offline — the config text never leaves the machine.
func (a *App) AnalyzeFCZoning(configText string) fczone.Report {
	return fczone.Parse(configText)
}
