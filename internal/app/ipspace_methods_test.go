package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppIPSpaceCreateAddUnlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	a := New()
	a.ipPath = path
	require.NoError(t, a.IPSpaceCreate("pw"))
	require.True(t, a.IPSpaceStatus().Unlocked)

	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)

	// A fresh controller unlocks the same file and sees the block.
	b := New()
	b.ipPath = path
	require.NoError(t, b.IPSpaceUnlock("pw"))
	tree := b.IPSpaceTree()
	require.Len(t, tree, 1)
	require.Equal(t, "Corp", tree[0].Record.Name)
}

func TestAppIPSpaceWrongPassphrase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.enc")
	a := New()
	a.ipPath = path
	require.NoError(t, a.IPSpaceCreate("pw"))

	b := New()
	b.ipPath = path
	require.Error(t, b.IPSpaceUnlock("nope"))
	require.False(t, b.IPSpaceStatus().Unlocked)
}
