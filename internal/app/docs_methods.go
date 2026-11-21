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
	data, err := cli.RenderDocsPDF()
	if err != nil {
		return "", err
	}
	path, err := wruntime.SaveFileDialog(a.lookupCtx(), wruntime.SaveDialogOptions{
		DefaultFilename: "meridian-tool-reference.pdf",
		Title:           "Save the tool reference as PDF",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
