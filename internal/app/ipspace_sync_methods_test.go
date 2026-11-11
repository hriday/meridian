package app

import (
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

func treeNames(nodes []ipspace.TreeNode) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Record.Name)
	}
	return out
}
