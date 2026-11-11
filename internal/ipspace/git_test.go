package ipspace

import (
	"context"
	"path/filepath"
	"testing"

	git "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
)

// bareRepo creates an empty bare repository on disk and returns its path, usable
// as a git URL via go-git's local transport — so the full clone/commit/push/
// non-fast-forward flow is exercised offline, with no network and no auth.
func bareRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	_, err := git.PlainInit(dir, true)
	require.NoError(t, err)
	return dir
}

func gitB(url string) GitBackend {
	return GitBackend{Cfg: GitConfig{URL: url, Branch: "main", Path: "ipspace.enc"}, Label: "git"}
}

func TestGitGetEmptyRepoIsNil(t *testing.T) {
	b := gitB(bareRepo(t))
	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Nil(t, blob)
	require.Empty(t, ver)
}

func TestGitFirstPushThenGet(t *testing.T) {
	b := gitB(bareRepo(t))
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))

	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("v1"), blob)
	require.NotEmpty(t, ver)
}

func TestGitUpdatePush(t *testing.T) {
	b := gitB(bareRepo(t))
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	_, v1, _ := b.Get(context.Background())
	require.NoError(t, b.Put(context.Background(), []byte("v2"), v1))
	blob, _, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("v2"), blob)
}

// A stale push (someone else committed in between) is a non-fast-forward and is
// refused as ErrRemoteChanged — the real git CAS.
func TestGitStalePushIsRemoteChanged(t *testing.T) {
	b := gitB(bareRepo(t))
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	_, v1, _ := b.Get(context.Background())
	require.NoError(t, b.Put(context.Background(), []byte("v2"), v1)) // moves HEAD on
	err := b.Put(context.Background(), []byte("v1-edit"), v1)         // stale base
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestGitVerifyEmptyRepoRejected(t *testing.T) {
	require.Error(t, gitB(bareRepo(t)).Verify(context.Background()))
}

func TestGitVerifyNonEmptyOK(t *testing.T) {
	b := gitB(bareRepo(t))
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	require.NoError(t, b.Verify(context.Background()))
}

func TestGitPullPushViaBackend(t *testing.T) {
	b := gitB(bareRepo(t))
	require.NoError(t, Push(context.Background(), b, "team", sampleEvents(), ""))
	got, ver, err := Pull(context.Background(), b, "team")
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NotEmpty(t, ver)
}
