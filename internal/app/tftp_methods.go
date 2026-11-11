package app

import (
	"errors"
	"os"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/tftp"
)

// TftpView is the receive-only TFTP grab's state for the frontend.
type TftpView struct {
	Running bool        `json:"running"`
	Port    int         `json:"port"`
	Files   []tftp.File `json:"files"`
}

// TftpStart binds the TFTP receiver (0 → OS-chosen port); starting while running
// restarts it. A bind failure is returned.
func (a *App) TftpStart(port int) error { return a.tftpSrv.Start(port) }

// TftpStop stops the receiver and clears its files.
func (a *App) TftpStop() { a.tftpSrv.Stop() }

// TftpList returns the running state, port, and received files.
func (a *App) TftpList() TftpView {
	running, port, files := a.tftpSrv.List()
	return TftpView{Running: running, Port: port, Files: files}
}

// TftpPreview returns a text (or hex) preview of a received file.
func (a *App) TftpPreview(id int) string {
	s, _ := a.tftpSrv.Preview(id, 256*1024)
	return s
}

// TftpSave writes a received file to a path the user chooses, returning the
// saved path (empty if cancelled).
func (a *App) TftpSave(id int) (string, error) {
	content, ok := a.tftpSrv.Content(id)
	if !ok {
		return "", errors.New("file no longer available")
	}
	name := "upload"
	for _, f := range a.TftpList().Files {
		if f.ID == id {
			name = f.Name
			break
		}
	}
	path, err := wruntime.SaveFileDialog(a.lookupCtx(), wruntime.SaveDialogOptions{
		DefaultFilename: name,
		Title:           "Save received file",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
