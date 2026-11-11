package ipspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalBackendGetMissingIsNil(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	blob, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Nil(t, blob, "a not-yet-created remote reads as nil, not an error")
}

func TestLocalBackendPutGetRoundTrip(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "sub", "remote.enc")}
	require.NoError(t, b.Put(context.Background(), []byte("hello")))
	blob, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), blob)
}

func TestLocalBackendPutIsAtomic(t *testing.T) {
	// A Put over an existing file leaves no stray temp files beside it.
	dir := t.TempDir()
	b := LocalBackend{Path: filepath.Join(dir, "remote.enc")}
	require.NoError(t, b.Put(context.Background(), []byte("v1")))
	require.NoError(t, b.Put(context.Background(), []byte("v2")))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "remote.enc", entries[0].Name())
}

func TestPushThenPull(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	ev := sampleEvents()
	require.NoError(t, Push(context.Background(), b, "team secret", ev))

	got, err := Pull(context.Background(), b, "team secret")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "10.1.0.0/16", got[1].Record.Prefix)
}

func TestPullMissingRemoteIsNil(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	got, err := Pull(context.Background(), b, "team secret")
	require.NoError(t, err)
	require.Nil(t, got, "pulling a remote that does not exist yet yields no events, no error")
}

func TestPullWrongPassphrase(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	require.NoError(t, Push(context.Background(), b, "right", sampleEvents()))
	_, err := Pull(context.Background(), b, "wrong")
	require.Error(t, err)
}

func TestLocalBackendDescribe(t *testing.T) {
	b := LocalBackend{Path: "/Volumes/share/ipspace.enc"}
	require.Contains(t, b.Describe(), "/Volumes/share/ipspace.enc")
}
