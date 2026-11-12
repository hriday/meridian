package app

import (
	"fmt"
	"os"
	"path/filepath"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/ipspace"
)

// Importing a hand-off file is directed, never automatic: open + decrypt the
// file, preview it, then choose a destination — merge into an existing
// inventory, land it as a new one, or just look (read-only, discarded). The
// decrypted file is held in a.pendingImport between the open and the chosen
// action; nothing is written until the user confirms.

// ImportInfo describes an opened hand-off file for the destination chooser.
type ImportInfo struct {
	Name    string `json:"name"`    // the inventory's name in the file
	Mode    string `json:"mode"`    // "park" | "share"
	Count   int    `json:"count"`   // records in the file
	HasSync bool   `json:"hasSync"` // park files carry a backend config
}

func (a *App) importInfo() ImportInfo {
	hf := a.pendingImport
	recs := ipspace.FromEvents("", hf.Events).Records()
	return ImportInfo{Name: hf.Name, Mode: hf.Mode, Count: len(recs), HasSync: hf.Sync != nil}
}

// setPendingImport decrypts a hand-off blob and stashes it for a destination
// choice. Assumes the lock is held.
func (a *App) setPendingImport(blob []byte, pass string) (ImportInfo, error) {
	hf, err := openHandoff(blob, pass)
	if err != nil {
		return ImportInfo{}, err
	}
	a.pendingImport = hf
	return a.importInfo(), nil
}

// IPSpaceImportOpen lets the user pick a .enc file, decrypts it with the hand-off
// password, and returns a preview. Nothing is written yet.
func (a *App) IPSpaceImportOpen(pass string) (ImportInfo, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	path, err := wruntime.OpenFileDialog(a.lookupCtx(), wruntime.OpenDialogOptions{
		Title: "Import an inventory file",
	})
	if err != nil || path == "" {
		return ImportInfo{}, err
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		return ImportInfo{}, err
	}
	return a.setPendingImport(blob, pass)
}

// IPSpaceImportCancel discards an opened-but-unapplied import.
func (a *App) IPSpaceImportCancel() {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.pendingImport = nil
}

// IPSpaceImportTree returns the opened file's record tree for a read-only "just
// look" (nothing is saved).
func (a *App) IPSpaceImportTree() []ipspace.TreeNode {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if a.pendingImport == nil {
		return nil
	}
	return ipspace.FromEvents("", a.pendingImport.Events).Tree()
}

// targetLive returns the live store/pass/path of an unlocked inventory (the
// active one, or a parked unlocked one), or ok=false if it must be unlocked first.
func (a *App) targetLive(id string) (store *ipspace.Store, pass, path string, ok bool) {
	if id == a.activeID && a.ipUnlocked {
		return a.ipStore, a.ipPass, a.ipPath, true
	}
	if live, ok := a.invLive[id]; ok && live.unlocked {
		return live.store, live.pass, filepath.Join(a.invDir(id), "store.enc"), true
	}
	return nil, "", "", false
}

// mergeImport reconciles the opened file against a target inventory. apply=false
// previews; apply=true writes the merge when there is no conflict. A hand-off has
// no shared sync base, so the merge is against a nil base (event-ID union):
// records the file adds come in, same-id records edited on both sides surface as
// conflicts, and shared history dedupes.
func (a *App) mergeImport(targetID string, apply bool) (SyncResult, error) {
	if a.pendingImport == nil {
		return SyncResult{}, fmt.Errorf("open a file to import first")
	}
	store, pass, path, ok := a.targetLive(targetID)
	if !ok {
		return SyncResult{}, fmt.Errorf("unlock that inventory first, then import into it")
	}
	res := ipspace.Merge(nil, store.Events(), a.pendingImport.Events)
	if !apply || len(res.Conflicts) > 0 {
		return toSyncResult(res, false), nil
	}
	merged := ipspace.FromEvents(ipspace.Actor(), res.Merged)
	if err := ipspace.Save(path, pass, res.Merged); err != nil {
		return SyncResult{}, err
	}
	if targetID == a.activeID {
		a.ipStore = merged
	} else {
		a.invLive[targetID].store = merged
	}
	a.pendingImport = nil
	return toSyncResult(res, true), nil
}

// IPSpaceImportPreviewMerge shows the diff of merging the opened file into an
// inventory, without writing.
func (a *App) IPSpaceImportPreviewMerge(targetID string) (SyncResult, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	return a.mergeImport(targetID, false)
}

// IPSpaceImportApplyMerge merges the opened file into an inventory and writes it
// (unless records conflict, in which case it writes nothing and returns them).
func (a *App) IPSpaceImportApplyMerge(targetID string) (SyncResult, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	return a.mergeImport(targetID, true)
}

// IPSpaceImportAsNew lands the opened file as its own new inventory (a free slot
// required), sealed with a new store passphrase. A park file also restores its
// sync config.
func (a *App) IPSpaceImportAsNew(name, passphrase string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	if a.pendingImport == nil {
		return fmt.Errorf("open a file to import first")
	}
	hf := a.pendingImport
	if err := a.createInventory(name, passphrase); err != nil {
		return err
	}
	a.ipStore = ipspace.FromEvents(ipspace.Actor(), hf.Events)
	if err := a.ipSave(); err != nil {
		return err
	}
	if hf.Sync != nil {
		if err := a.saveSyncConfig(*hf.Sync); err != nil {
			return err
		}
	}
	a.pendingImport = nil
	return nil
}
