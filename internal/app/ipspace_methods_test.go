package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppIPSpaceCreateAddUnlock(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Acme", "pw"))
	require.True(t, a.IPSpaceStatus().Unlocked)

	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)

	// A fresh controller on the same root loads the registry and unlocks the inventory.
	b := New()
	b.invRoot = a.invRoot
	b.IPSpaceList() // loads registry + active
	require.NoError(t, b.IPSpaceUnlock("pw"))
	tree := b.IPSpaceTree()
	require.Len(t, tree, 1)
	require.Equal(t, "Corp", tree[0].Record.Name)
}

func TestAppIPSpaceWrongPassphrase(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Acme", "pw"))

	b := New()
	b.invRoot = a.invRoot
	b.IPSpaceList()
	require.Error(t, b.IPSpaceUnlock("nope"))
	require.False(t, b.IPSpaceStatus().Unlocked)
}

func TestInventoryCreateSwitchLimit(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Acme", "pw1"))
	require.NoError(t, a.IPSpaceCreateInventory("Globex", "pw2"))
	require.NoError(t, a.IPSpaceCreateInventory("Initech", "pw3"))
	require.Len(t, a.IPSpaceList(), 3)

	// Fourth is refused.
	require.Error(t, a.IPSpaceCreateInventory("Umbrella", "pw4"))

	// The newest is active and unlocked; the first is parked (unlocked state kept).
	ids := a.IPSpaceList()
	require.Equal(t, "Initech", active(ids).Name)

	first := ids[0].ID
	require.NoError(t, a.IPSpaceSetActive(first))
	require.Equal(t, first, a.IPSpaceActive())
	// Switching back to an inventory unlocked earlier keeps it unlocked.
	require.True(t, a.IPSpaceStatus().Unlocked)
}

func TestInventoryRenameRemove(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("Acme", "pw1"))
	require.NoError(t, a.IPSpaceCreateInventory("Globex", "pw2"))
	globex := a.IPSpaceActive()

	require.NoError(t, a.IPSpaceRenameInventory(globex, "Globex Inc"))
	require.Equal(t, "Globex Inc", find(a.IPSpaceList(), globex).Name)

	require.NoError(t, a.IPSpaceRemoveInventory(globex))
	require.Len(t, a.IPSpaceList(), 1)
	require.NotEqual(t, globex, a.IPSpaceActive(), "removing the active inventory selects another")

	// A fresh controller sees only the remaining inventory.
	b := New()
	b.invRoot = a.invRoot
	require.Len(t, b.IPSpaceList(), 1)
}

func active(list []InvInfo) InvInfo {
	for _, i := range list {
		if i.Active {
			return i
		}
	}
	return InvInfo{}
}
func find(list []InvInfo, id string) InvInfo {
	for _, i := range list {
		if i.ID == id {
			return i
		}
	}
	return InvInfo{}
}
