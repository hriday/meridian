package ipspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalBackendGetMissingIsNil(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Nil(t, blob, "a not-yet-created remote reads as nil, not an error")
	require.Empty(t, ver, "a missing remote has the empty version")
}

func TestLocalBackendPutGetRoundTrip(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "sub", "remote.enc")}
	require.NoError(t, b.Put(context.Background(), []byte("hello"), ""))
	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), blob)
	require.NotEmpty(t, ver, "a present remote has a non-empty version")
}

func TestLocalBackendPutIsAtomic(t *testing.T) {
	// A Put over an existing file leaves no stray temp files beside it.
	dir := t.TempDir()
	b := LocalBackend{Path: filepath.Join(dir, "remote.enc")}
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	_, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.NoError(t, b.Put(context.Background(), []byte("v2"), ver))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "remote.enc", entries[0].Name())
}

// CAS: a Put whose expected version no longer matches the remote is refused, so
// a stale writer can't clobber a change it never saw.
func TestLocalBackendPutStaleVersionRefused(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	_, v1, err := b.Get(context.Background())
	require.NoError(t, err)

	// A second writer moves the remote forward.
	require.NoError(t, b.Put(context.Background(), []byte("v2"), v1))

	// The first writer, still holding v1, is now stale → refused, remote untouched.
	err = b.Put(context.Background(), []byte("v1-edit"), v1)
	require.ErrorIs(t, err, ErrRemoteChanged)
	blob, _, _ := b.Get(context.Background())
	require.Equal(t, []byte("v2"), blob, "the refused Put left the remote as the winner wrote it")
}

// CAS on a first push: expecting an empty remote but finding one already there
// is refused (someone else pushed first).
func TestLocalBackendPutExpectedEmptyButPresent(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	require.NoError(t, b.Put(context.Background(), []byte("theirs"), ""))
	err := b.Put(context.Background(), []byte("mine"), "")
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestPushThenPull(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	ev := sampleEvents()
	require.NoError(t, Push(context.Background(), b, "team secret", ev, ""))

	got, ver, err := Pull(context.Background(), b, "team secret")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NotEmpty(t, ver)
	require.Equal(t, "10.1.0.0/16", got[1].Record.Prefix)
}

func TestPullMissingRemoteIsNil(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	got, ver, err := Pull(context.Background(), b, "team secret")
	require.NoError(t, err)
	require.Nil(t, got, "pulling a remote that does not exist yet yields no events, no error")
	require.Empty(t, ver)
}

func TestPullWrongPassphrase(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	require.NoError(t, Push(context.Background(), b, "right", sampleEvents(), ""))
	_, _, err := Pull(context.Background(), b, "wrong")
	require.Error(t, err)
}

func TestPushStaleRefused(t *testing.T) {
	b := LocalBackend{Path: filepath.Join(t.TempDir(), "remote.enc")}
	require.NoError(t, Push(context.Background(), b, "pw", sampleEvents(), ""))
	_, ver, _ := Pull(context.Background(), b, "pw")
	require.NoError(t, Push(context.Background(), b, "pw", sampleEvents(), ver)) // moves it forward
	err := Push(context.Background(), b, "pw", sampleEvents(), ver)              // stale
	require.True(t, errors.Is(err, ErrRemoteChanged))
}

func TestLocalBackendDescribe(t *testing.T) {
	b := LocalBackend{Path: "/Volumes/share/ipspace.enc"}
	require.Contains(t, b.Describe(), "/Volumes/share/ipspace.enc")
}
