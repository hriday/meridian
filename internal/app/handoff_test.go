package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSealOpenHandoffParkRoundTrip(t *testing.T) {
	a := newUnlockedStore(t, "team")
	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	require.NoError(t, a.IPSpaceSyncConfigureLocal("/Volumes/share/ipspace.enc"))

	blob, name, err := a.sealHandoff("park", "handoff-pw")
	require.NoError(t, err)
	require.Contains(t, name, "park")
	require.Contains(t, name, ".enc")

	hf, err := openHandoff(blob, "handoff-pw")
	require.NoError(t, err)
	require.Equal(t, "park", hf.Mode)
	require.Equal(t, "Test", hf.Name)
	require.Len(t, hf.Events, 1)
	require.NotNil(t, hf.Sync, "park carries the sync config so re-import is fully wired")
	require.Equal(t, "local", hf.Sync.Kind)
}

func TestSealHandoffShareExcludesCredentials(t *testing.T) {
	a := newUnlockedStore(t, "team")
	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	require.NoError(t, a.IPSpaceSyncConfigureLocal("/Volumes/share/ipspace.enc"))

	blob, _, err := a.sealHandoff("share", "hp")
	require.NoError(t, err)

	hf, err := openHandoff(blob, "hp")
	require.NoError(t, err)
	require.Equal(t, "share", hf.Mode)
	require.Len(t, hf.Events, 1)
	require.Nil(t, hf.Sync, "share must not carry the sync config / credentials")
}

func TestOpenHandoffWrongPassword(t *testing.T) {
	a := newUnlockedStore(t, "team")
	_, err := a.IPSpaceAdd("10.0.0.0/8", "Corp", "active", nil, "")
	require.NoError(t, err)
	blob, _, err := a.sealHandoff("share", "right")
	require.NoError(t, err)
	_, err = openHandoff(blob, "wrong")
	require.Error(t, err)
}

func TestSealHandoffValidation(t *testing.T) {
	a := newUnlockedStore(t, "team")
	_, _, err := a.sealHandoff("bogus", "pw")
	require.Error(t, err)
	_, _, err = a.sealHandoff("share", "   ")
	require.Error(t, err)
}

func TestSanitizeFilename(t *testing.T) {
	require.Equal(t, "Acme-Corp", sanitizeFilename("Acme Corp"))
	require.Equal(t, "a-b", sanitizeFilename("a/b"))
	require.Equal(t, "inventory", sanitizeFilename("  //  "))
}
