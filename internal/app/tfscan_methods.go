package app

import "github.com/hriday/meridian/internal/tfscan"

// AnalyzeTerraform reads a Terraform config offline (raw HCL .tf, or a
// `terraform show -json` plan) and returns what the infrastructure exposes to
// the internet, plus findings and an exposure graph. source is "hcl", "json",
// or "" to auto-detect.
func (a *App) AnalyzeTerraform(text, source string) (tfscan.Analysis, error) {
	return tfscan.Analyze(text, source)
}
