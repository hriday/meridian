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
	// A session-imported recipe is keyed by its file path.
	if path, ok := a.importedRunbooks[profile]; ok {
		return cli.RunRunbookFile(path, target)
	}
	d, ok := cli.RunRunbookData(profile, target)
	if !ok {
		return cli.RunbookData{}, fmt.Errorf("unknown runbook %q", profile)
	}
	return d, nil
}

// ImportRunbook opens a file picker for a user-defined recipe (YAML / TOML /
// JSON), validates it, remembers it for this session, and returns its summary so
// the GUI can add it to the profile list. Returns an empty info (no error) if
// the user cancels.
func (a *App) ImportRunbook() (cli.RunbookInfo, error) {
	path, err := wruntime.OpenFileDialog(a.lookupCtx(), wruntime.OpenDialogOptions{
		Title: "Import a runbook recipe",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Runbook recipes (*.yaml, *.yml, *.toml, *.json)", Pattern: "*.yaml;*.yml;*.toml;*.json"},
		},
	})
	if err != nil || path == "" {
		return cli.RunbookInfo{}, err
	}
	info, err := cli.RunbookFileInfo(path)
	if err != nil {
		return cli.RunbookInfo{}, err
	}
	if a.importedRunbooks == nil {
		a.importedRunbooks = map[string]string{}
	}
	info.ID = path // key the profile by its path so RunRunbook can find it
	a.importedRunbooks[path] = path
	return info, nil
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
