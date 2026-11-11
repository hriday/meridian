package ipspace

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
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

// The REAL concurrent-write race (not just the curHead early-check): two clones
// at the same HEAD, one pushes, the other's push is a genuine non-fast-forward
// rejected by go-git — and mapPushErr must turn that into ErrRemoteChanged.
func TestGitRealNonFastForwardMapsToRemoteChanged(t *testing.T) {
	url := bareRepo(t)
	require.NoError(t, gitB(url).Put(context.Background(), []byte("v1"), "")) // seed main

	r1 := cloneMain(t, url)
	r2 := cloneMain(t, url) // both at the same HEAD

	// r1 commits and pushes — advances the remote.
	require.NoError(t, commitAndPushMain(r1, []byte("a")))
	// r2 commits on the now-stale HEAD and pushes — a real non-fast-forward.
	err := commitAndPushMain(r2, []byte("b"))
	require.Error(t, err, "go-git must reject the stale push")
	require.ErrorIs(t, mapPushErr(err), ErrRemoteChanged, "the real push rejection must map to ErrRemoteChanged")
}

func TestSanitizeURLStripsCredentials(t *testing.T) {
	require.Equal(t, "https://github.com/acme/ipam.git", sanitizeURL("https://user:ghp_token@github.com/acme/ipam.git"))
	require.Equal(t, "https://github.com/acme/ipam.git", sanitizeURL("https://github.com/acme/ipam.git"))
	require.NotContains(t, (GitConfig{URL: "https://u:secret@h/r.git", Path: "p"}).Describe(), "secret")
}

func cloneMain(t *testing.T, url string) *git.Repository {
	t.Helper()
	r, err := git.CloneContext(context.Background(), memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL: url, ReferenceName: plumbing.NewBranchReferenceName("main"), SingleBranch: true,
	})
	require.NoError(t, err)
	return r
}

func commitAndPushMain(r *git.Repository, data []byte) error {
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	f, err := wt.Filesystem.Create("ipspace.enc")
	if err != nil {
		return err
	}
	io.Copy(f, bytes.NewReader(data))
	f.Close()
	if _, err := wt.Add("ipspace.enc"); err != nil {
		return err
	}
	if _, err := wt.Commit("edit", &git.CommitOptions{Author: &object.Signature{Name: "t", Email: "t@t", When: time.Now()}}); err != nil {
		return err
	}
	return r.Push(&git.PushOptions{
		RefSpecs: []config.RefSpec{config.RefSpec("refs/heads/main:refs/heads/main")},
	})
}
