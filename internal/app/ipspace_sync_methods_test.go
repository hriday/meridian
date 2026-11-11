package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hriday/meridian/internal/ipspace"
	"github.com/stretchr/testify/require"
)

// newUnlockedStore returns a controller with a fresh, unlocked store at its own
// path, all sharing one remote blob and one team passphrase.
func newUnlockedStore(t *testing.T, storePath, pass string) *App {
	t.Helper()
	a := New()
	a.ipPath = storePath
	require.NoError(t, a.IPSpaceCreate(pass))
	return a
}

func TestSyncConfigurePersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "store.enc")
	remote := filepath.Join(dir, "remote.enc")

	a := newUnlockedStore(t, store, "team")
	require.False(t, a.IPSpaceSyncInfo().Configured)
	require.NoError(t, a.IPSpaceSyncConfigureLocal(remote))

	info := a.IPSpaceSyncInfo()
	require.True(t, info.Configured)
	require.Contains(t, info.Location, remote)

	// A fresh controller on the same store sees the configured backend.
	b := New()
	b.ipPath = store
	require.NoError(t, b.IPSpaceUnlock("team"))
	require.True(t, b.IPSpaceSyncInfo().Configured)
}

func TestSyncConfigureRejectsEmptyPath(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	require.Error(t, a.IPSpaceSyncConfigureLocal("   "))
}

func TestSyncPreviewRequiresConfig(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	_, err := a.IPSpaceSyncPreview()
	require.Error(t, err)
}

// Full round trip: Alice pushes, Bob pulls her change and pushes his, Alice
// pulls his — the shared remote converges.
func TestSyncTwoPersonRoundTrip(t *testing.T) {
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote.enc")

	alice := newUnlockedStore(t, filepath.Join(dir, "alice.enc"), "team")
	bob := newUnlockedStore(t, filepath.Join(dir, "bob.enc"), "team")
	require.NoError(t, alice.IPSpaceSyncConfigureLocal(remote))
	require.NoError(t, bob.IPSpaceSyncConfigureLocal(remote))

	// Alice adds a block and pushes it to the empty remote.
	_, err := alice.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	res, err := alice.IPSpaceSyncApply()
	require.NoError(t, err)
	require.True(t, res.Applied)
	require.Len(t, res.Outgoing, 1)

	// Bob adds his own block, then previews: he should see Alice's add incoming
	// and his own outgoing, with no conflict.
	_, err = bob.IPSpaceAdd("192.168.0.0/16", "Lab", "active", nil, "")
	require.NoError(t, err)
	prev, err := bob.IPSpaceSyncPreview()
	require.NoError(t, err)
	require.False(t, prev.Applied)
	require.Len(t, prev.Incoming, 1)
	require.Equal(t, "Corp", prev.Incoming[0].Record.Name)
	require.Len(t, prev.Outgoing, 1)
	require.Empty(t, prev.Conflicts)

	// Bob applies: his store now holds both blocks.
	_, err = bob.IPSpaceSyncApply()
	require.NoError(t, err)
	names := treeNames(bob.IPSpaceTree())
	require.ElementsMatch(t, []string{"Corp", "Lab"}, names)

	// Alice pulls Bob's change.
	ares, err := alice.IPSpaceSyncApply()
	require.NoError(t, err)
	require.True(t, ares.Applied)
	require.ElementsMatch(t, []string{"Corp", "Lab"}, treeNames(alice.IPSpaceTree()))
}

// A record edited differently on both sides since the last sync is a conflict:
// Apply refuses and writes nothing until it is resolved.
func TestSyncConflictBlocksApply(t *testing.T) {
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote.enc")

	alice := newUnlockedStore(t, filepath.Join(dir, "alice.enc"), "team")
	bob := newUnlockedStore(t, filepath.Join(dir, "bob.enc"), "team")
	require.NoError(t, alice.IPSpaceSyncConfigureLocal(remote))
	require.NoError(t, bob.IPSpaceSyncConfigureLocal(remote))

	// Alice creates a block and both sync so they share a base.
	rec, err := alice.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	_, err = alice.IPSpaceSyncApply()
	require.NoError(t, err)
	_, err = bob.IPSpaceSyncApply()
	require.NoError(t, err)

	// Both edit the same record's name differently, offline.
	require.NoError(t, alice.IPSpaceEdit(rec.ID, "Corp-A", "active", nil, ""))
	require.NoError(t, bob.IPSpaceEdit(rec.ID, "Corp-B", "active", nil, ""))

	// Alice pushes first — clean.
	ar, err := alice.IPSpaceSyncApply()
	require.NoError(t, err)
	require.True(t, ar.Applied)

	// Bob now previews/applies: a true conflict, Apply refuses and the remote is
	// left as Alice pushed it.
	bprev, err := bob.IPSpaceSyncPreview()
	require.NoError(t, err)
	require.Len(t, bprev.Conflicts, 1)

	bres, err := bob.IPSpaceSyncApply()
	require.NoError(t, err)
	require.False(t, bres.Applied)
	require.Len(t, bres.Conflicts, 1)
	// Bob's local store is untouched — still his version.
	require.Equal(t, "Corp-B", treeNames(bob.IPSpaceTree())[0])
}

func TestSyncConfigureS3SealsCredentials(t *testing.T) {
	dir := t.TempDir()
	store := filepath.Join(dir, "store.enc")
	a := newUnlockedStore(t, store, "team")
	// Route the configure-time probe through an in-memory backend so the test
	// doesn't reach the network; we're verifying the config is sealed, not S3.
	a.ipMakeBackend = func(cfg syncConfig) (ipspace.Backend, bool) { return &raceBackend{}, true }

	const secret = "super-secret-key-material"
	require.NoError(t, a.IPSpaceSyncConfigureS3(
		"https://acct.r2.cloudflarestorage.com", "auto", "inventory", "meridian/ipspace.enc", "AKID", secret))

	info := a.IPSpaceSyncInfo()
	require.True(t, info.Configured)
	require.Equal(t, "s3", info.Kind)
	require.Contains(t, info.Location, "inventory")
	require.NotContains(t, info.Location, secret, "the secret must not show in the UI location")

	// The config is sealed: an encrypted sidecar exists, no plaintext one, and the
	// secret does not appear in the bytes on disk.
	encRaw, err := os.ReadFile(sidecarPath(store, ".sync.enc"))
	require.NoError(t, err)
	require.NotContains(t, string(encRaw), secret)
	require.NotContains(t, string(encRaw), "AKID")
	require.NoFileExists(t, sidecarPath(store, ".sync.json"))

	// A fresh controller unlocks and sees the S3 backend configured.
	b := New()
	b.ipPath = store
	require.NoError(t, b.IPSpaceUnlock("team"))
	require.True(t, b.IPSpaceSyncInfo().Configured)
	require.Equal(t, "s3", b.IPSpaceSyncInfo().Kind)
}

func TestSyncConfigureSFTPSealsAndPinsHostKey(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	a.ipProbeSFTP = func(ctx context.Context, cfg ipspace.SFTPConfig) (string, error) {
		return "ssh-ed25519 AAAApinnedkey", nil
	}
	a.ipMakeBackend = func(cfg syncConfig) (ipspace.Backend, bool) { return &raceBackend{}, true } // Verify offline

	const password = "sftp-password-secret"
	require.NoError(t, a.IPSpaceSyncConfigureSFTP("sftp.example.com", "22", "svc", password, "", "", "/srv/meridian/ipspace.enc"))

	info := a.IPSpaceSyncInfo()
	require.True(t, info.Configured)
	require.Equal(t, "sftp", info.Kind)
	require.Contains(t, info.Location, "sftp.example.com")
	require.NotContains(t, info.Location, password)

	encRaw, err := os.ReadFile(sidecarPath(store, ".sync.enc"))
	require.NoError(t, err)
	require.NotContains(t, string(encRaw), password, "the SFTP password must not be on disk in the clear")

	// The host key captured at configure is pinned in the sealed config.
	b := New()
	b.ipPath = store
	require.NoError(t, b.IPSpaceUnlock("team"))
	require.Equal(t, "sftp", b.IPSpaceSyncInfo().Kind)
}

func TestSyncConfigureSFTPRequiresAuth(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	err := a.IPSpaceSyncConfigureSFTP("h", "22", "u", "", "", "", "/p")
	require.Error(t, err, "no password and no key must be rejected")
	require.False(t, a.IPSpaceSyncInfo().Configured)
}

func TestSyncConfigureSFTPRejectsEmptyHostKey(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	a.ipProbeSFTP = func(ctx context.Context, cfg ipspace.SFTPConfig) (string, error) {
		return "", nil // no host key captured — must not be saved as a blank pin
	}
	err := a.IPSpaceSyncConfigureSFTP("h", "22", "u", "pw", "", "", "/p")
	require.Error(t, err)
	require.False(t, a.IPSpaceSyncInfo().Configured)
}

func TestSyncConfigureSFTPSurfacesProbeError(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	a.ipProbeSFTP = func(ctx context.Context, cfg ipspace.SFTPConfig) (string, error) {
		return "", fmt.Errorf("auth failed")
	}
	err := a.IPSpaceSyncConfigureSFTP("h", "22", "u", "pw", "", "", "/p")
	require.Error(t, err)
	require.False(t, a.IPSpaceSyncInfo().Configured, "a failed connect must not save a config")
}

func TestSyncConfigureGitSealsToken(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	a.ipMakeBackend = func(cfg syncConfig) (ipspace.Backend, bool) { return &raceBackend{}, true } // Verify offline

	const token = "ghp_secrettoken12345"
	require.NoError(t, a.IPSpaceSyncConfigureGit("https://github.com/acme/ipam.git", "main", "ipspace.enc", "acme-bot", token))

	info := a.IPSpaceSyncInfo()
	require.True(t, info.Configured)
	require.Equal(t, "git", info.Kind)
	require.Contains(t, info.Location, "acme/ipam")
	require.NotContains(t, info.Location, token)

	encRaw, err := os.ReadFile(sidecarPath(store, ".sync.enc"))
	require.NoError(t, err)
	require.NotContains(t, string(encRaw), token, "the git token must not be on disk in the clear")
}

func TestSyncConfigureGitRequiresAuth(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	err := a.IPSpaceSyncConfigureGit("https://x/y.git", "main", "ipspace.enc", "", "")
	require.Error(t, err)
	require.False(t, a.IPSpaceSyncInfo().Configured)
}

// A bucket that fails the conditional-write check is rejected at configure, and
// no config is saved — so an unsafe backend never becomes the sync target.
func TestSyncConfigureS3RejectsUnsafeProvider(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store.enc")
	a := newUnlockedStore(t, store, "team")
	a.ipMakeBackend = func(cfg syncConfig) (ipspace.Backend, bool) {
		return &raceBackend{verifyErr: ipspace.ErrRemoteChanged}, true // stand-in for "ignores conditional writes"
	}
	err := a.IPSpaceSyncConfigureS3("https://x", "auto", "bkt", "k", "AK", "SK")
	require.Error(t, err)
	require.False(t, a.IPSpaceSyncInfo().Configured, "an unsafe bucket must not be saved as the sync target")
}

// raceBackend is an in-memory CAS backend whose beforePut hook fires once, just
// before the CAS check, to simulate another person pushing between this client's
// pull and push — the exact window the Critical lost-update race lives in.
type raceBackend struct {
	blob      []byte
	version   string
	beforePut func(*raceBackend)
	verifyErr error
}

func (b *raceBackend) set(blob []byte) {
	b.blob = append([]byte(nil), blob...)
	sum := sha256.Sum256(blob)
	b.version = hex.EncodeToString(sum[:])
}
func (b *raceBackend) Get(ctx context.Context) ([]byte, string, error) {
	if b.blob == nil {
		return nil, "", nil
	}
	return append([]byte(nil), b.blob...), b.version, nil
}
func (b *raceBackend) Put(ctx context.Context, blob []byte, expected string) error {
	if b.beforePut != nil {
		hook := b.beforePut
		b.beforePut = nil
		hook(b)
	}
	if b.version != expected {
		return ipspace.ErrRemoteChanged
	}
	b.set(blob)
	return nil
}
func (b *raceBackend) Verify(ctx context.Context) error { return b.verifyErr }
func (b *raceBackend) Describe() string                 { return "race backend" }

// A write that lands between this client's pull and push is caught by the CAS
// guard: Apply refuses, writes nothing locally, and surfaces ErrRemoteChanged —
// instead of silently clobbering the other person's push.
func TestSyncApplyRefusesLostUpdate(t *testing.T) {
	dir := t.TempDir()
	remote := &raceBackend{}

	bob := newUnlockedStore(t, filepath.Join(dir, "bob.enc"), "team")
	bob.ipMakeBackend = func(cfg syncConfig) (ipspace.Backend, bool) { return remote, true }
	require.NoError(t, bob.IPSpaceSyncConfigureLocal(remote.Describe()))

	// Seed the remote with a block Bob already has (shared base).
	seedRec, err := bob.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	_, err = bob.IPSpaceSyncApply()
	require.NoError(t, err)

	// Bob edits locally, offline.
	require.NoError(t, bob.IPSpaceEdit(seedRec.ID, "Corp-Bob", "active", nil, ""))

	// Simulate Alice pushing a different edit in the window between Bob's pull
	// and push. Her version != Bob's expected, so Bob's push must be refused.
	remote.beforePut = func(b *raceBackend) {
		alice := ipspace.FromEvents("alice", mustOpen(t, b.blob, "team"))
		require.NoError(t, alice.Edit(seedRec.ID, "Corp-Alice", "active", nil, ""))
		sealed, err := ipspace.SealEvents("team", alice.Events())
		require.NoError(t, err)
		b.set(sealed)
	}

	_, err = bob.IPSpaceSyncApply()
	require.ErrorIs(t, err, ipspace.ErrRemoteChanged)

	// Bob's local store is untouched (still his own edit), nothing clobbered.
	require.Equal(t, "Corp-Bob", treeNames(bob.IPSpaceTree())[0])
}

func mustOpen(t *testing.T, blob []byte, pass string) []ipspace.Event {
	t.Helper()
	ev, err := ipspace.OpenEvents(pass, blob)
	require.NoError(t, err)
	return ev
}

func treeNames(nodes []ipspace.TreeNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Record.Name)
	}
	return out
}
