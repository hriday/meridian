package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// exportShare builds a share hand-off blob from a one-record inventory.
func exportBlob(t *testing.T, mode, prefix, name, pass string) []byte {
	t.Helper()
	src := newUnlockedStore(t, "team")
	_, err := src.IPSpaceAdd(prefix, name, "active", nil, "")
	require.NoError(t, err)
	if mode == "park" {
		require.NoError(t, src.IPSpaceSyncConfigureLocal("/Volumes/share/ipspace.enc"))
	}
	blob, _, err := src.sealHandoff(mode, pass)
	require.NoError(t, err)
	return blob
}

func TestImportMergeIntoActive(t *testing.T) {
	blob := exportBlob(t, "share", "10.0.0.0/8", "Corp", "hp")

	dst := newUnlockedStore(t, "team")
	_, err := dst.IPSpaceAdd("192.168.0.0/16", "Lab", "active", nil, "")
	require.NoError(t, err)

	info, err := dst.setPendingImport(blob, "hp")
	require.NoError(t, err)
	require.Equal(t, 1, info.Count)
	require.Equal(t, "share", info.Mode)

	prev, err := dst.mergeImport(dst.activeID, false)
	require.NoError(t, err)
	require.False(t, prev.Applied)
	require.Len(t, prev.Incoming, 1)
	require.Equal(t, "Corp", prev.Incoming[0].Record.Name)

	res, err := dst.mergeImport(dst.activeID, true)
	require.NoError(t, err)
	require.True(t, res.Applied)
	require.ElementsMatch(t, []string{"Corp", "Lab"}, treeNames(dst.IPSpaceTree()))
	require.Nil(t, dst.pendingImport, "a successful merge clears the pending import")
}

func TestImportMergeConflictWritesNothing(t *testing.T) {
	// Both inventories independently record the SAME prefix as different records
	// — that's a same-prefix duplicate, surfaced but not a hard conflict. To force
	// a real conflict we reuse the same record id via a round-trip.
	src := newUnlockedStore(t, "team")
	rec, err := src.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	blob, _, err := src.sealHandoff("share", "hp")
	require.NoError(t, err)

	// dst imports it as a new inventory, then both edit the same record id.
	dst := newUnlockedStore(t, "team")
	_, err = dst.setPendingImport(blob, "hp")
	require.NoError(t, err)
	require.NoError(t, dst.IPSpaceImportAsNew("Imported", "pw2345"))
	require.NoError(t, dst.IPSpaceEdit(rec.ID, "Corp-local", "active", nil, ""))

	// A second file edits the same record differently.
	require.NoError(t, src.IPSpaceEdit(rec.ID, "Corp-remote", "active", nil, ""))
	blob2, _, err := src.sealHandoff("share", "hp")
	require.NoError(t, err)
	_, err = dst.setPendingImport(blob2, "hp")
	require.NoError(t, err)

	res, err := dst.mergeImport(dst.activeID, true)
	require.NoError(t, err)
	require.False(t, res.Applied, "a conflict blocks the merge")
	require.Len(t, res.Conflicts, 1)
	require.Equal(t, "Corp-local", treeNames(dst.IPSpaceTree())[0], "local is untouched")
	require.NotNil(t, dst.pendingImport, "the import stays open to resolve")
}

func TestImportAsNewInventory(t *testing.T) {
	blob := exportBlob(t, "park", "10.0.0.0/8", "Corp", "hp")

	dst := newUnlockedStore(t, "team")
	_, err := dst.setPendingImport(blob, "hp")
	require.NoError(t, err)
	require.NoError(t, dst.IPSpaceImportAsNew("Acme (imported)", "pw2345"))

	require.Len(t, dst.IPSpaceList(), 2)
	require.Equal(t, "Acme (imported)", active(dst.IPSpaceList()).Name)
	require.ElementsMatch(t, []string{"Corp"}, treeNames(dst.IPSpaceTree()))
	// A park file restores the sync config into the new inventory.
	require.True(t, dst.IPSpaceSyncInfo().Configured)
	require.Nil(t, dst.pendingImport)
}

func TestImportJustLookSavesNothing(t *testing.T) {
	blob := exportBlob(t, "share", "10.0.0.0/8", "Corp", "hp")

	dst := newUnlockedStore(t, "team")
	_, err := dst.IPSpaceAdd("192.168.0.0/16", "Lab", "active", nil, "")
	require.NoError(t, err)
	before := treeNames(dst.IPSpaceTree())

	_, err = dst.setPendingImport(blob, "hp")
	require.NoError(t, err)
	look := dst.IPSpaceImportTree()
	require.Len(t, look, 1)
	require.Equal(t, "Corp", look[0].Record.Name)

	// Looking saves nothing: the active inventory is unchanged, and cancel clears it.
	require.ElementsMatch(t, before, treeNames(dst.IPSpaceTree()))
	dst.IPSpaceImportCancel()
	require.Nil(t, dst.pendingImport)
	require.Len(t, dst.IPSpaceList(), 1)
}

func TestImportWrongPassword(t *testing.T) {
	blob := exportBlob(t, "share", "10.0.0.0/8", "Corp", "right")
	dst := newUnlockedStore(t, "team")
	_, err := dst.setPendingImport(blob, "wrong")
	require.Error(t, err)
}
