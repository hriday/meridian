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

// syncConfig is the backend configuration persisted (encrypted) beside the store.
// It may hold backend credentials (S3), so it is sealed with the store passphrase
// rather than written in the clear.
type syncConfig struct {
	Kind string             `json:"kind"`           // "local" | "s3"
	Path string             `json:"path,omitempty"` // local/mounted path
	S3   *ipspace.S3Config  `json:"s3,omitempty"`   // S3-compatible connection + credentials
}

// errNoBackend means no sync backend is configured (distinct from a configured
// backend that failed to build).
var errNoBackend = fmt.Errorf("set up a sync location first")

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

func (a *App) syncConfigEncPath() string    { return sidecarPath(a.ipPath, ".sync.enc") }
func (a *App) syncConfigLegacyPath() string { return sidecarPath(a.ipPath, ".sync.json") }
func (a *App) syncBasePath() string         { return sidecarPath(a.ipPath, ".base.enc") }

// sidecarPath turns ".../ipspace.enc" into ".../ipspace<suffix>".
func sidecarPath(storePath, suffix string) string {
	return strings.TrimSuffix(storePath, ".enc") + suffix
}

// loadSyncConfig returns the backend config, decrypting the sealed sidecar with
// the store passphrase. It also reads a pre-encryption plaintext sidecar (local
// path only) so a config written by an earlier build keeps working.
func (a *App) loadSyncConfig() (syncConfig, bool) {
	if raw, err := os.ReadFile(a.syncConfigEncPath()); err == nil {
		if !a.ipUnlocked {
			return syncConfig{}, false
		}
		plain, err := ipspace.Open(a.ipPass, raw)
		if err != nil {
			return syncConfig{}, false
		}
		var cfg syncConfig
		if json.Unmarshal(plain, &cfg) != nil || cfg.Kind == "" {
			return syncConfig{}, false
		}
		return cfg, true
	}
	// Legacy plaintext local-path config from before the config was encrypted.
	if raw, err := os.ReadFile(a.syncConfigLegacyPath()); err == nil {
		var cfg syncConfig
		if json.Unmarshal(raw, &cfg) == nil && cfg.Path != "" {
			cfg.Kind = "local"
			return cfg, true
		}
	}
	return syncConfig{}, false
}

func (a *App) saveSyncConfig(cfg syncConfig) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	blob, err := ipspace.Seal(a.ipPass, raw)
	if err != nil {
		return err
	}
	if err := os.WriteFile(a.syncConfigEncPath(), blob, 0o600); err != nil {
		return err
	}
	os.Remove(a.syncConfigLegacyPath()) // drop any superseded plaintext config
	return nil
}

// backend builds the configured backend, or errNoBackend if none is set.
func (a *App) backend() (ipspace.Backend, error) {
	cfg, ok := a.loadSyncConfig()
	if !ok {
		return nil, errNoBackend
	}
	if a.ipMakeBackend != nil {
		b, ok := a.ipMakeBackend(cfg)
		if !ok {
			return nil, errNoBackend
		}
		return b, nil
	}
	switch cfg.Kind {
	case "local":
		return ipspace.LocalBackend{Path: cfg.Path}, nil
	case "s3":
		if cfg.S3 == nil {
			return nil, errNoBackend
		}
		return ipspace.NewS3Backend(context.Background(), *cfg.S3)
	default:
		return nil, errNoBackend
	}
}

// IPSpaceSyncInfo reports whether a sync backend is configured and where. It
// never includes credentials (Describe is bucket/key/endpoint or a path).
func (a *App) IPSpaceSyncInfo() SyncInfo {
	cfg, ok := a.loadSyncConfig()
	if !ok {
		return SyncInfo{}
	}
	loc := cfg.Path
	if b, err := a.backend(); err == nil {
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
	return a.saveSyncConfig(syncConfig{Kind: "local", Path: path})
}

// IPSpaceSyncConfigureS3 points sync at an object in an S3-compatible bucket
// (AWS S3, Cloudflare R2, Backblaze B2, MinIO). Credentials are sealed with the
// store passphrase, never written in the clear.
func (a *App) IPSpaceSyncConfigureS3(endpoint, region, bucket, key, accessKey, secret string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	cfg := ipspace.S3Config{
		Endpoint:  strings.TrimSpace(endpoint),
		Region:    strings.TrimSpace(region),
		Bucket:    strings.TrimSpace(bucket),
		Key:       strings.TrimSpace(key),
		AccessKey: strings.TrimSpace(accessKey),
		Secret:    secret,
	}
	if cfg.Bucket == "" || cfg.Key == "" {
		return fmt.Errorf("enter the bucket and object key")
	}
	if _, err := ipspace.NewS3Backend(context.Background(), cfg); err != nil {
		return err
	}
	return a.saveSyncConfig(syncConfig{Kind: "s3", S3: &cfg})
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
	b, err = a.backend()
	if err != nil {
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
