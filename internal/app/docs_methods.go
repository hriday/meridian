package app

import (
	"os"

	"github.com/hriday/meridian/internal/cli"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// SaveDocsPDF renders the full tool reference to a PDF in Go and saves it via a
// Save dialog, returning the chosen path (or "" if cancelled). The GUI uses this
// instead of browser print-to-PDF, which is a no-op in the macOS webview.
func (a *App) SaveDocsPDF() (string, error) {
	// Ask for the destination first, so cancelling costs nothing to render.
	path, err := wruntime.SaveFileDialog(a.lookupCtx(), wruntime.SaveDialogOptions{
		DefaultFilename: "meridian-tool-reference.pdf",
		Title:           "Save the tool reference as PDF",
	})
	if err != nil || path == "" {
		return "", err
	}
	data, err := cli.RenderDocsPDF()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// GUIReference returns the GUI Tools overview — every tool as the app shows it,
// grouped like the sidebar, with a plain-English description. It's the GUI
// counterpart to CLICommands and backs the in-app Tools overview page.
func (a *App) GUIReference() []cli.GUIGroup {
	return cli.GUICatalog()
}

// SaveGUIReferencePDF renders the Tools overview to a PDF in Go and saves it via
// a Save dialog, returning the chosen path (or "" if cancelled).
func (a *App) SaveGUIReferencePDF() (string, error) {
	// Ask for the destination first, so cancelling costs nothing to render.
	path, err := wruntime.SaveFileDialog(a.lookupCtx(), wruntime.SaveDialogOptions{
		DefaultFilename: "meridian-tools-overview.pdf",
		Title:           "Save the tools overview as PDF",
	})
	if err != nil || path == "" {
		return "", err
	}
	data, err := cli.RenderGUIReferencePDF()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
