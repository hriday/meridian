package app

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/ipspace"
)

// A hand-off file is an inventory exported to a single encrypted blob you can
// carry — hand to a colleague, or park to free a slot and restore later. It is
// sealed with a HAND-OFF PASSWORD chosen at export and entered at import, kept
// separate from any store passphrase so it can pass between people who share no
// store. Two modes: "park" carries the sync config (credentials) so re-import
// restores the inventory fully wired, for your own round-trip; "share" carries
// the inventory data only — no credentials — for handing to someone else.
const handoffFormat = "meridian-ipspace-handoff/1"

type handoffFile struct {
	Format string          `json:"format"`
	Mode   string          `json:"mode"` // "park" | "share"
	Name   string          `json:"name"`
	Events []ipspace.Event `json:"events"`
	Sync   *syncConfig     `json:"sync,omitempty"` // park only
}

var filenameUnsafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeFilename(name string) string {
	s := filenameUnsafe.ReplaceAllString(strings.TrimSpace(name), "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		s = "inventory"
	}
	return s
}

func (a *App) activeNameInternal() string {
	if m := a.findInv(a.activeID); m != nil {
		return m.Name
	}
	return "inventory"
}

// sealHandoff builds and seals an export of the active inventory. Assumes the
// IP-space lock is held. park includes the sync config (credentials); share does
// not. Returns the sealed blob and a suggested filename.
func (a *App) sealHandoff(mode, pass string) (blob []byte, filename string, err error) {
	if err = a.requireUnlocked(); err != nil {
		return
	}
	if mode != "park" && mode != "share" {
		err = fmt.Errorf("choose park or share")
		return
	}
	if strings.TrimSpace(pass) == "" {
		err = fmt.Errorf("set a password to seal the file")
		return
	}
	hf := handoffFile{
		Format: handoffFormat,
		Mode:   mode,
		Name:   a.activeNameInternal(),
		Events: a.ipStore.Events(),
	}
	if mode == "park" {
		if cfg, ok := a.loadSyncConfig(); ok {
			c := cfg
			hf.Sync = &c
		}
	}
	raw, err := json.Marshal(hf)
	if err != nil {
		return
	}
	blob, err = ipspace.Seal(pass, raw)
	filename = sanitizeFilename(hf.Name) + "-" + mode + ".enc"
	return
}

// openHandoff decrypts and parses a hand-off blob (used by import).
func openHandoff(blob []byte, pass string) (*handoffFile, error) {
	raw, err := ipspace.Open(pass, blob)
	if err != nil {
		return nil, err
	}
	var hf handoffFile
	if err := json.Unmarshal(raw, &hf); err != nil {
		return nil, fmt.Errorf("this file isn't a Meridian inventory export")
	}
	if hf.Format != handoffFormat {
		return nil, fmt.Errorf("this file isn't a Meridian inventory export")
	}
	return &hf, nil
}

// IPSpaceExportHandoff seals the active inventory and writes it to a file the
// user picks. mode is "park" (full, incl. sync settings — for yourself) or
// "share" (data only, no credentials — for someone else).
func (a *App) IPSpaceExportHandoff(mode, pass string) (string, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	blob, filename, err := a.sealHandoff(mode, pass)
	if err != nil {
		return "", err
	}
	path, err := wruntime.SaveFileDialog(a.lookupCtx(), wruntime.SaveDialogOptions{
		DefaultFilename: filename,
		Title:           "Export inventory",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
