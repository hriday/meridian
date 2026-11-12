package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hriday/meridian/internal/ipspace"
)

// IP space holds up to maxInventories separate, isolated inventories — e.g. one
// per company. Each lives in its own directory under invRoot with its own store,
// passphrase, sync config and base; they never co-mingle. A registry (index.json)
// lists them and remembers which is active. The active inventory's live state
// lives in the a.ip* fields (so every store/sync method operates on the active
// one); other unlocked inventories are parked in invLive until switched back.
const maxInventories = 3

type invMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type registry struct {
	Inventories []invMeta `json:"inventories"`
	Active      string    `json:"active"`
}

// invLive is an unlocked inventory's in-memory state, parked while another is active.
type invLive struct {
	pass          string
	store         *ipspace.Store
	unlocked      bool
	syncCfg       *syncConfig
	syncCfgLoaded bool
}

// InvInfo is one inventory as the UI sees it.
type InvInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Unlocked bool   `json:"unlocked"`
	HasSync  bool   `json:"hasSync"`
}

func defaultInvRoot() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "Meridian", "inventories")
}

func newInvID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (a *App) registryPath() string  { return filepath.Join(a.invRoot, "index.json") }
func (a *App) invDir(id string) string { return filepath.Join(a.invRoot, id) }
func (a *App) activeDir() string      { return a.invDir(a.activeID) }

// ensureInv lazily loads the registry (migrating a legacy single store the first
// time) so tests can point invRoot at a temp dir before anything touches disk.
func (a *App) ensureInv() {
	if a.invLoaded {
		return
	}
	a.invLoaded = true
	if a.invLive == nil {
		a.invLive = map[string]*invLive{}
	}
	a.migrateLegacy()
	a.loadRegistry()
}

func (a *App) loadRegistry() {
	a.invs = nil
	a.activeID = ""
	raw, err := os.ReadFile(a.registryPath())
	if err != nil {
		return
	}
	var reg registry
	if json.Unmarshal(raw, &reg) != nil {
		return
	}
	a.invs = reg.Inventories
	a.activeID = reg.Active
	if a.findInv(a.activeID) == nil && len(a.invs) > 0 {
		a.activeID = a.invs[0].ID
	}
	a.applyActivePath()
}

func (a *App) saveRegistry() error {
	if err := os.MkdirAll(a.invRoot, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(registry{Inventories: a.invs, Active: a.activeID}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.registryPath(), raw, 0o600)
}

// migrateLegacy moves a pre-multi-inventory single store (…/Meridian/ipspace.enc
// and its sidecars) into a first inventory, once, before the registry exists.
func (a *App) migrateLegacy() {
	if _, err := os.Stat(a.registryPath()); err == nil {
		return // registry already exists
	}
	legacy := filepath.Join(filepath.Dir(a.invRoot), "ipspace.enc")
	if !ipspace.Exists(legacy) {
		return
	}
	id := newInvID()
	dir := a.invDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	// Move the store (the critical data) first. If that fails, abort with the
	// legacy store untouched so the next launch retries — never a half-migration.
	newStore := filepath.Join(dir, "store.enc")
	if err := os.Rename(legacy, newStore); err != nil {
		return
	}
	// Sidecars are best-effort — losing a sync config or base is degraded, not
	// data loss, and they move with the same suffix scheme.
	for suffix, name := range map[string]string{".sync.enc": "store.sync.enc", ".base.enc": "store.base.enc", ".sync.json": "store.sync.json"} {
		if from := sidecarPath(legacy, suffix); ipspace.Exists(from) {
			_ = os.Rename(from, filepath.Join(dir, name))
		}
	}
	a.invs = []invMeta{{ID: id, Name: "My IP space"}}
	a.activeID = id
	if err := a.saveRegistry(); err != nil {
		// Couldn't record the migration — roll the store back so it isn't orphaned
		// under a random-id dir, and retry on the next launch.
		_ = os.Rename(newStore, legacy)
		a.invs = nil
		a.activeID = ""
	}
}

func (a *App) findInv(id string) *invMeta {
	for i := range a.invs {
		if a.invs[i].ID == id {
			return &a.invs[i]
		}
	}
	return nil
}

// applyActivePath points the active-inventory live fields' store path at the
// active inventory's directory.
func (a *App) applyActivePath() {
	if a.activeID == "" {
		a.ipPath = ""
		return
	}
	a.ipPath = filepath.Join(a.activeDir(), "store.enc")
}

// parkActive saves the active inventory's unlocked live state so switching back
// to it later doesn't re-prompt for the passphrase.
func (a *App) parkActive() {
	if a.activeID == "" {
		return
	}
	if a.ipUnlocked {
		a.invLive[a.activeID] = &invLive{
			pass: a.ipPass, store: a.ipStore, unlocked: true,
			syncCfg: a.ipSyncCfg, syncCfgLoaded: a.ipSyncCfgLoaded,
		}
	} else {
		delete(a.invLive, a.activeID)
	}
}

// loadActiveLive restores the target inventory's parked live state, or a locked
// state if it was never unlocked (or is parked no more).
func (a *App) loadActiveLive(id string) {
	if live, ok := a.invLive[id]; ok {
		a.ipPass, a.ipStore, a.ipUnlocked = live.pass, live.store, live.unlocked
		a.ipSyncCfg, a.ipSyncCfgLoaded = live.syncCfg, live.syncCfgLoaded
		delete(a.invLive, id) // it's the active one now
		return
	}
	a.ipPass, a.ipStore, a.ipUnlocked = "", nil, false
	a.ipSyncCfg, a.ipSyncCfgLoaded = nil, false
}

// IPSpaceList returns the inventories for the selector.
func (a *App) IPSpaceList() []InvInfo {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	out := make([]InvInfo, 0, len(a.invs))
	for _, m := range a.invs {
		unlocked := m.ID == a.activeID && a.ipUnlocked
		if live, ok := a.invLive[m.ID]; ok {
			unlocked = live.unlocked
		}
		out = append(out, InvInfo{
			ID: m.ID, Name: m.Name, Active: m.ID == a.activeID,
			Unlocked: unlocked, HasSync: a.inventoryHasSync(m.ID),
		})
	}
	return out
}

// inventoryHasSync reports whether an inventory has a sync config on disk
// (without needing it unlocked — just whether the sealed sidecar exists).
func (a *App) inventoryHasSync(id string) bool {
	store := filepath.Join(a.invDir(id), "store.enc")
	return ipspace.Exists(sidecarPath(store, ".sync.enc")) || ipspace.Exists(sidecarPath(store, ".sync.json"))
}

// IPSpaceActive returns the active inventory id.
func (a *App) IPSpaceActive() string {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	return a.activeID
}

// IPSpaceSetActive switches the active inventory (parking the current one's
// unlocked state so a later switch back doesn't re-prompt).
func (a *App) IPSpaceSetActive(id string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	if a.findInv(id) == nil {
		return fmt.Errorf("no such inventory")
	}
	if id == a.activeID {
		return nil
	}
	a.parkActive()
	a.activeID = id
	a.applyActivePath()
	a.loadActiveLive(id)
	return a.saveRegistry()
}

// IPSpaceCreateInventory adds a new, empty inventory and makes it active +
// unlocked. Up to maxInventories.
func (a *App) IPSpaceCreateInventory(name, passphrase string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	return a.createInventory(name, passphrase)
}

// createInventory adds a new, empty inventory and makes it active + unlocked.
// Assumes the lock is held and ensureInv has run.
func (a *App) createInventory(name, passphrase string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("give the inventory a name")
	}
	if len(a.invs) >= maxInventories {
		return fmt.Errorf("all %d inventory slots are in use — park one to a file first", maxInventories)
	}
	id := newInvID()
	dir := a.invDir(id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	store := filepath.Join(dir, "store.enc")
	if err := ipspace.Create(store, passphrase); err != nil {
		return err
	}
	a.parkActive()
	a.invs = append(a.invs, invMeta{ID: id, Name: name})
	a.activeID = id
	a.ipPath = store
	a.ipPass = passphrase
	a.ipStore = ipspace.NewStore(ipspace.Actor())
	a.ipUnlocked = true
	a.ipSyncCfg, a.ipSyncCfgLoaded = nil, false
	return a.saveRegistry()
}

// IPSpaceRenameInventory renames an inventory.
func (a *App) IPSpaceRenameInventory(id, name string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("give the inventory a name")
	}
	m := a.findInv(id)
	if m == nil {
		return fmt.Errorf("no such inventory")
	}
	m.Name = name
	return a.saveRegistry()
}

// IPSpaceRemoveInventory deletes an inventory and its files. This is
// irreversible; the caller confirms first.
func (a *App) IPSpaceRemoveInventory(id string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	if a.findInv(id) == nil {
		return fmt.Errorf("no such inventory")
	}
	// Delete the files first; only drop it from the registry once they're gone,
	// so a failed delete leaves a consistent state rather than a dangling entry.
	if err := os.RemoveAll(a.invDir(id)); err != nil {
		return err
	}
	filtered := a.invs[:0:0]
	for _, m := range a.invs {
		if m.ID != id {
			filtered = append(filtered, m)
		}
	}
	a.invs = filtered
	delete(a.invLive, id)
	if a.activeID == id {
		a.activeID = ""
		if len(a.invs) > 0 {
			a.activeID = a.invs[0].ID
		}
		a.applyActivePath()
		a.loadActiveLive(a.activeID)
	}
	return a.saveRegistry()
}
