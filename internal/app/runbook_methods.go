package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/cli"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

// RunbookRender formats an already-run runbook as "json", "html", or markdown
// ("md"/"txt") — for export, without re-running the tools.
func (a *App) RunbookRender(d cli.RunbookData, format string) string {
	return cli.RenderRunbook(d, format)
}

// SaveReport opens a Save dialog seeded with defaultName and writes content to
// the chosen path, returning it (or "" if the user cancelled). Used by the GUI
// to export a runbook report as text/JSON/HTML.
func (a *App) SaveReport(defaultName, content string) (string, error) {
	path, err := wruntime.SaveFileDialog(a.lookupCtx(), wruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "Save report",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
