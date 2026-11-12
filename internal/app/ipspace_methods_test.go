package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hriday/meridian/internal/ipspace"
	"github.com/stretchr/testify/require"
)

// A pre-multi-inventory single store migrates into the first inventory, intact.
func TestLegacyStoreMigrates(t *testing.T) {
	meridian := filepath.Join(t.TempDir(), "Meridian")
	require.NoError(t, os.MkdirAll(meridian, 0o700))
	legacy := filepath.Join(meridian, "ipspace.enc")
	st := ipspace.NewStore("tester")
	_, err := st.Add("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	require.NoError(t, ipspace.Save(legacy, "pw1234", st.Events()))

	a := New()
	a.invRoot = filepath.Join(meridian, "inventories")
	list := a.IPSpaceList()
	require.Len(t, list, 1)
	require.Equal(t, "My IP space", list[0].Name)
	require.NoFileExists(t, legacy, "the legacy store is moved, not copied")

	require.NoError(t, a.IPSpaceUnlock("pw1234"))
	tree := a.IPSpaceTree()
	require.Len(t, tree, 1)
	require.Equal(t, "Corp", tree[0].Record.Name)
}

// Concurrent binding calls (Wails dispatches each on its own goroutine) must not
// race the invLive map or the ip* live state. Run under -race.
func TestInventoryConcurrentAccessNoRace(t *testing.T) {
	a := New()
	a.invRoot = t.TempDir()
	require.NoError(t, a.IPSpaceCreateInventory("A", "pw1aaa"))
	require.NoError(t, a.IPSpaceCreateInventory("B", "pw2bbb"))
	ids := a.IPSpaceList()

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(4)
		go func() { defer wg.Done(); a.IPSpaceList() }()
		go func() { defer wg.Done(); _ = a.IPSpaceSetActive(ids[0].ID) }()
		go func() { defer wg.Done(); _ = a.IPSpaceSetActive(ids[1].ID) }()
		go func() { defer wg.Done(); a.IPSpaceStatus() }()
	}
	wg.Wait()
	require.Len(t, a.IPSpaceList(), 2)
}

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
