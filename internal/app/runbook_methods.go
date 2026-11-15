package app

import (
	"fmt"
	"strings"

	"github.com/hriday/meridian/internal/cli"
)

// RunbookProfiles lists the available composite audit profiles.
func (a *App) RunbookProfiles() []cli.RunbookInfo { return cli.Runbooks() }

// RunRunbook runs a composite profile (e.g. "audit-domain", "triage-host")
// against target and returns the per-step results. Each step is an existing tool
// run with the same engine as the GUI/CLI. It can take a while (a full path
// trace is one of the steps).
func (a *App) RunRunbook(profile, target string) (cli.RunbookData, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return cli.RunbookData{}, fmt.Errorf("enter a target")
	}
	d, ok := cli.RunRunbookData(profile, target)
	if !ok {
		return cli.RunbookData{}, fmt.Errorf("unknown runbook %q", profile)
	}
	return d, nil
}
