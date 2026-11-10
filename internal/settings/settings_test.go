package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func useTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := configDir
	configDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { configDir = old })
	return dir
}

func TestAtlasKeyMissingFileEmpty(t *testing.T) {
	useTempDir(t)
	k, err := AtlasKey()
	require.NoError(t, err)
	require.Equal(t, "", k)
}

func TestSetAndGetAtlasKey(t *testing.T) {
	useTempDir(t)
	require.NoError(t, SetAtlasKey("abc"))
	k, err := AtlasKey()
	require.NoError(t, err)
	require.Equal(t, "abc", k)
}

func TestSetAtlasKeyFilePerms(t *testing.T) {
	dir := useTempDir(t)
	require.NoError(t, SetAtlasKey("secret"))
	info, err := os.Stat(filepath.Join(dir, "meridian", "settings.json"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "settings file must be owner-only")
}

func TestClearAtlasKey(t *testing.T) {
	useTempDir(t)
	require.NoError(t, SetAtlasKey("abc"))
	require.NoError(t, ClearAtlasKey())
	k, err := AtlasKey()
	require.NoError(t, err)
	require.Equal(t, "", k)
}

func TestSetAtlasKeyTrims(t *testing.T) {
	useTempDir(t)
	require.NoError(t, SetAtlasKey("  k \n"))
	k, _ := AtlasKey()
	require.Equal(t, "k", k)
}
