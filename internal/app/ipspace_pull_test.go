package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPullRecordsAcrossInventories(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Source", "pw1aaa"))
	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", []string{"prod"}, "note")
	require.NoError(t, err)
	_, err = a.IPSpaceAdd("172.16.0.0/12", "Lab", "active", nil, "")
	require.NoError(t, err)
	srcID := a.IPSpaceActive()

	require.NoError(t, a.IPSpaceCreateInventory("Dest", "pw2bbb")) // Dest active, Source parked unlocked

	// The source's tree is readable for choosing what to pull.
	require.Len(t, a.IPSpaceInventoryTree(srcID), 2)

	rep, err := a.IPSpacePullRecords(srcID, []string{"10.0.0.0/8"})
	require.NoError(t, err)
	require.Equal(t, 1, rep.Added)
	require.Empty(t, rep.Skipped)
	require.ElementsMatch(t, []string{"Corp"}, treeNames(a.IPSpaceTree()))

	// Pulling the same prefix again is skipped, not overwritten.
	rep2, err := a.IPSpacePullRecords(srcID, []string{"10.0.0.0/8"})
	require.NoError(t, err)
	require.Equal(t, 0, rep2.Added)
	require.Len(t, rep2.Skipped, 1)

	// The source is untouched.
	require.ElementsMatch(t, []string{"Corp", "Lab"}, treeNames(a.IPSpaceInventoryTree(srcID)))
}

func TestPullFromSelfRejected(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Only", "pw1aaa"))
	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	_, err = a.IPSpacePullRecords(a.IPSpaceActive(), []string{"10.0.0.0/8"})
	require.Error(t, err)
}

func TestPullFromLockedRejected(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Source", "pw1aaa"))
	srcID := a.IPSpaceActive()
	require.NoError(t, a.IPSpaceCreateInventory("Dest", "pw2bbb"))

	// Re-open in a fresh controller: Source is present but locked (not in invLive).
	b := New()
	b.invRoot = a.invRoot
	b.IPSpaceList()
	require.NoError(t, b.IPSpaceUnlock("pw2bbb")) // only the active (Dest) is unlocked
	require.Nil(t, b.IPSpaceInventoryTree(srcID), "a locked inventory's tree isn't exposed")
	_, err := b.IPSpacePullRecords(srcID, []string{"10.0.0.0/8"})
	require.Error(t, err)
}
