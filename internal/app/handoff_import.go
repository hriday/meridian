package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Opened  bool   `json:"opened"`  // false when the user cancelled the file picker
	Name    string `json:"name"`    // the inventory's name in the file
	Mode    string `json:"mode"`    // "park" | "share"
	Count   int    `json:"count"`   // records in the file
	HasSync bool   `json:"hasSync"` // park files carry a backend config
}

func (a *App) importInfo() ImportInfo {
	hf := a.pendingImport
	recs := ipspace.FromEvents("", hf.Events).Records()
	return ImportInfo{Opened: true, Name: hf.Name, Mode: hf.Mode, Count: len(recs), HasSync: hf.Sync != nil}
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

// commonEventsByID returns the events of a whose id also appears in b, in a's
// causal order — the shared history of the two logs, used as the merge base.
func commonEventsByID(a, b []ipspace.Event) []ipspace.Event {
	inB := make(map[string]bool, len(b))
	for _, e := range b {
		inB[e.ID] = true
	}
	var out []ipspace.Event
	for _, e := range a {
		if inB[e.ID] {
			out = append(out, e)
		}
	}
	return out
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
// previews; apply=true writes the merge when there is no conflict. The base is
// the events the two logs SHARE by id (their common history), so this is a true
// 3-way merge: a record the file added/edited/DELETED since that shared point
// shows as incoming (a delete the file made is applied and previewed, not
// silently dropped), a change made only locally is outgoing, and a record
// changed on both sides — including edit-vs-delete — is a conflict that blocks
// the write. A share from someone with no shared history has an empty base, so
// everything in it is incoming.
func (a *App) mergeImport(targetID string, apply bool) (SyncResult, error) {
	if a.pendingImport == nil {
		return SyncResult{}, fmt.Errorf("open a file to import first")
	}
	store, pass, path, ok := a.targetLive(targetID)
	if !ok {
		return SyncResult{}, fmt.Errorf("unlock that inventory first, then import into it")
	}
	local := store.Events()
	base := commonEventsByID(local, a.pendingImport.Events)
	res := ipspace.Merge(base, local, a.pendingImport.Events)
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
// sync config. The inventory's files are written in full BEFORE it is committed
// to the registry and made active, so a failed write can't leave a half-created
// active inventory; a failure at the final registry write rolls everything back.
func (a *App) IPSpaceImportAsNew(name, passphrase string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	if a.pendingImport == nil {
		return fmt.Errorf("open a file to import first")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("give the inventory a name")
	}
	if len(a.invs) >= maxInventories {
		return fmt.Errorf("all %d inventory slots are in use — park one to a file first", maxInventories)
	}
	if len(passphrase) < 6 {
		return fmt.Errorf("use a passphrase of at least 6 characters")
	}
	hf := a.pendingImport
	id := newInvID()
	dir := a.invDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	storePath := filepath.Join(dir, "store.enc")
	// Write all of the inventory's files first; roll back the dir on any failure.
	if err := ipspace.Save(storePath, passphrase, hf.Events); err != nil {
		os.RemoveAll(dir)
		return err
	}
	if hf.Sync != nil {
		raw, err := json.Marshal(*hf.Sync)
		if err == nil {
			var blob []byte
			blob, err = ipspace.Seal(passphrase, raw)
			if err == nil {
				err = os.WriteFile(sidecarPath(storePath, ".sync.enc"), blob, 0o600)
			}
		}
		if err != nil {
			os.RemoveAll(dir)
			return err
		}
	}
	// Files are durable — now commit to the registry and make it active.
	prevActive := a.activeID
	a.parkActive()
	a.invs = append(a.invs, invMeta{ID: id, Name: name})
	a.activeID = id
	a.ipPath = storePath
	a.ipPass = passphrase
	a.ipStore = ipspace.FromEvents(ipspace.Actor(), hf.Events)
	a.ipUnlocked = true
	if hf.Sync != nil {
		c := *hf.Sync
		a.ipSyncCfg, a.ipSyncCfgLoaded = &c, true
	} else {
		a.ipSyncCfg, a.ipSyncCfgLoaded = nil, false
	}
	if err := a.saveRegistry(); err != nil {
		// Roll back to the previous active inventory and remove the new files.
		os.RemoveAll(dir)
		a.invs = a.invs[:len(a.invs)-1]
		a.activeID = prevActive
		a.applyActivePath()
		a.loadActiveLive(prevActive)
		return err
	}
	a.pendingImport = nil
	return nil
}
