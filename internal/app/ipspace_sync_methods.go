package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/ipspace"
)

// Sync lets several people share one IP-space inventory by hosting a single
// encrypted blob on a backend of their choice. Each person's changes stay local
// until they sync; a sync pulls the shared blob, merges it against the log as of
// their last sync (the "base"), surfaces the diff, and — when there is no
// conflict — adopts the merged log locally and pushes it back. The shared blob
// uses the same passphrase as the local store, so a team agrees on one inventory
// passphrase. Backend config and the base live in sidecars next to the store.

// syncConfig is the (non-secret) backend configuration persisted beside the store.
type syncConfig struct {
	Kind string `json:"kind"` // "local" today
	Path string `json:"path"`
}

// SyncInfo reports the configured backend to the UI.
type SyncInfo struct {
	Configured bool   `json:"configured"`
	Kind       string `json:"kind"`
	Location   string `json:"location"`
}

// SyncResult is the diff from a preview or the outcome of an apply.
type SyncResult struct {
	Incoming          []ipspace.Change          `json:"incoming"`          // changes only on the remote
	Outgoing          []ipspace.Change          `json:"outgoing"`          // changes only local
	Conflicts         []ipspace.Conflict        `json:"conflicts"`         // records changed on both sides
	DuplicatePrefixes []ipspace.PrefixCollision `json:"duplicatePrefixes"` // same prefix on >1 record after merge
	Applied           bool                      `json:"applied"`           // true when apply wrote the merge
}

func (a *App) syncConfigPath() string { return sidecarPath(a.ipPath, ".sync.json") }
func (a *App) syncBasePath() string   { return sidecarPath(a.ipPath, ".base.enc") }

// sidecarPath turns ".../ipspace.enc" into ".../ipspace<suffix>".
func sidecarPath(storePath, suffix string) string {
	return strings.TrimSuffix(storePath, ".enc") + suffix
}

func (a *App) loadSyncConfig() (syncConfig, bool) {
	raw, err := os.ReadFile(a.syncConfigPath())
	if err != nil {
		return syncConfig{}, false
	}
	var cfg syncConfig
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg.Path == "" {
		return syncConfig{}, false
	}
	return cfg, true
}

func (a *App) backend() (ipspace.Backend, bool) {
	cfg, ok := a.loadSyncConfig()
	if !ok {
		return nil, false
	}
	if a.ipMakeBackend != nil {
		return a.ipMakeBackend(cfg.Kind, cfg.Path)
	}
	switch cfg.Kind {
	case "local":
		return ipspace.LocalBackend{Path: cfg.Path}, true
	default:
		return nil, false
	}
}

// IPSpaceSyncInfo reports whether a sync backend is configured and where.
func (a *App) IPSpaceSyncInfo() SyncInfo {
	cfg, ok := a.loadSyncConfig()
	if !ok {
		return SyncInfo{}
	}
	b, ok := a.backend()
	loc := cfg.Path
	if ok {
		loc = b.Describe()
	}
	return SyncInfo{Configured: true, Kind: cfg.Kind, Location: loc}
}

// IPSpaceSyncConfigureLocal points sync at a shared file path (a mounted share,
// a synced folder, a USB drive).
func (a *App) IPSpaceSyncConfigureLocal(path string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("enter the shared file path to sync through")
	}
	cfg := syncConfig{Kind: "local", Path: path}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.syncConfigPath(), raw, 0o600)
}

// loadBase returns the event log as of the last successful sync, or nil for a
// first sync (no base yet).
func (a *App) loadBase() ([]ipspace.Event, error) {
	path := a.syncBasePath()
	if !ipspace.Exists(path) {
		return nil, nil
	}
	return ipspace.Load(path, a.ipPass)
}

func (a *App) saveBase(events []ipspace.Event) error {
	return ipspace.Save(a.syncBasePath(), a.ipPass, events)
}

// merge pulls the remote and reconciles it against the local log and the base.
// It returns the backend and the version it pulled so a following push can
// compare-and-swap against exactly what was merged.
func (a *App) merge() (res ipspace.MergeResult, b ipspace.Backend, version string, err error) {
	if err = a.requireUnlocked(); err != nil {
		return
	}
	var ok bool
	b, ok = a.backend()
	if !ok {
		err = fmt.Errorf("set up a sync location first")
		return
	}
	base, err := a.loadBase()
	if err != nil {
		return
	}
	remote, version, err := ipspace.Pull(context.Background(), b, a.ipPass)
	if err != nil {
		return
	}
	res = ipspace.Merge(base, a.ipStore.Events(), remote)
	return
}

// IPSpaceSyncPreview pulls the remote and shows the diff without writing anything.
func (a *App) IPSpaceSyncPreview() (SyncResult, error) {
	res, _, _, err := a.merge()
	if err != nil {
		return SyncResult{}, err
	}
	return toSyncResult(res, false), nil
}

// IPSpaceSyncApply pulls, merges, and — when there is no conflict — adopts the
// merged log locally, pushes it to the remote, and records it as the new base.
// When records were changed on both sides, it writes nothing and returns the
// conflicts so the user can resolve them first (never auto-clobbering a change).
// If someone else pushed between this pull and push, the backend refuses the
// write (ErrRemoteChanged) and nothing local is touched — the user reviews again.
func (a *App) IPSpaceSyncApply() (SyncResult, error) {
	res, b, version, err := a.merge()
	if err != nil {
		return SyncResult{}, err
	}
	if len(res.Conflicts) > 0 {
		return toSyncResult(res, false), nil
	}
	if err := ipspace.Push(context.Background(), b, a.ipPass, res.Merged, version); err != nil {
		return SyncResult{}, err
	}
	// Adopt the merged log locally and record it as the base for next time.
	a.ipStore = ipspace.FromEvents(ipspace.Actor(), res.Merged)
	if err := a.ipSave(); err != nil {
		return SyncResult{}, err
	}
	if err := a.saveBase(res.Merged); err != nil {
		return SyncResult{}, err
	}
	return toSyncResult(res, true), nil
}

func toSyncResult(r ipspace.MergeResult, applied bool) SyncResult {
	return SyncResult{
		Incoming:          r.Incoming,
		Outgoing:          r.Outgoing,
		Conflicts:         r.Conflicts,
		DuplicatePrefixes: r.DuplicatePrefixes,
		Applied:           applied,
	}
}
