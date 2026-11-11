package ipspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
)

// GitBackend stores the shared blob as a file in a git repository, committed and
// pushed on every sync. Git is a natural fit for the compare-and-swap: the
// version is the branch's commit hash, and a push that isn't a fast-forward of
// what was pulled is rejected by the server — surfaced as ErrRemoteChanged. The
// whole history of the inventory is kept as commits, for free. The remote repo
// must already exist (make an initial commit, e.g. add a README, before pointing
// Meridian at it).
type GitBackend struct {
	Cfg   GitConfig
	Label string
}

// GitConfig is the connection detail for a git remote. Secret fields (Token,
// PrivateKey, KeyPassphrase) are sealed with the store passphrase.
type GitConfig struct {
	URL           string // https://… or git@…:…
	Branch        string // default "main"
	Path          string // file path within the repo, e.g. "ipspace.enc"
	Username      string // for HTTPS token auth (default "git")
	Token         string // HTTPS personal access token / password
	PrivateKey    string // SSH PEM, optional
	KeyPassphrase string
}

func (c GitConfig) branch() string {
	if strings.TrimSpace(c.Branch) == "" {
		return "main"
	}
	return strings.TrimSpace(c.Branch)
}

// Describe is the credential-free label.
func (c GitConfig) Describe() string {
	return fmt.Sprintf("git %s (%s) [%s]", c.URL, c.branch(), c.Path)
}

func (c GitConfig) auth() (transport.AuthMethod, error) {
	if strings.TrimSpace(c.PrivateKey) != "" {
		return gitssh.NewPublicKeys("git", []byte(c.PrivateKey), c.KeyPassphrase)
	}
	if c.Token != "" {
		user := c.Username
		if user == "" {
			user = "git"
		}
		return &githttp.BasicAuth{Username: user, Password: c.Token}, nil
	}
	return nil, nil // local transport / no auth
}

// NewGitBackend builds a backend from config.
func NewGitBackend(c GitConfig) (*GitBackend, error) {
	if c.URL == "" || c.Path == "" {
		return nil, fmt.Errorf("git needs a repository URL and a file path")
	}
	return &GitBackend{Cfg: c, Label: c.Describe()}, nil
}

func (b GitBackend) refName() plumbing.ReferenceName {
	return plumbing.NewBranchReferenceName(b.Cfg.branch())
}

// cloneBranch clones the configured branch into memory. A truly empty remote
// returns (nil, nil) so the caller can create the first commit.
func (b GitBackend) cloneBranch(ctx context.Context) (*git.Repository, error) {
	auth, err := b.Cfg.auth()
	if err != nil {
		return nil, err
	}
	r, err := git.CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           b.Cfg.URL,
		Auth:          auth,
		ReferenceName: b.refName(),
		SingleBranch:  true,
	})
	if errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Get reads the blob file and returns the branch head as the version. A repo
// with commits but no blob file returns (nil, head, nil); a truly empty repo
// returns (nil, "", nil).
func (b GitBackend) Get(ctx context.Context) ([]byte, string, error) {
	r, err := b.cloneBranch(ctx)
	if err != nil {
		return nil, "", err
	}
	if r == nil {
		return nil, "", nil
	}
	head, err := r.Head()
	if err != nil {
		return nil, "", err
	}
	version := head.Hash().String()
	wt, err := r.Worktree()
	if err != nil {
		return nil, "", err
	}
	f, err := wt.Filesystem.Open(b.Cfg.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, version, nil
	}
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	blob, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}
	return blob, version, nil
}

// Put writes the blob file, commits, and pushes — but only if the branch head
// still equals expectedVersion (nothing pushed since the pull). A non-fast-
// forward push (someone pushed in between) maps to ErrRemoteChanged.
func (b GitBackend) Put(ctx context.Context, blob []byte, expectedVersion string) error {
	auth, err := b.Cfg.auth()
	if err != nil {
		return err
	}
	r, err := b.cloneBranch(ctx)
	if err != nil {
		return err
	}
	curHead := ""
	if r != nil {
		head, err := r.Head()
		if err != nil {
			return err
		}
		curHead = head.Hash().String()
	}
	if curHead != expectedVersion {
		return ErrRemoteChanged
	}
	if r == nil {
		// Empty remote: create the first commit and push it to the branch.
		return b.firstPush(ctx, auth, blob)
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	if err := writeWorktreeFile(wt, b.Cfg.Path, blob); err != nil {
		return err
	}
	if _, err := wt.Add(b.Cfg.Path); err != nil {
		return err
	}
	if _, err := wt.Commit("Update IP space inventory", &git.CommitOptions{Author: gitSignature()}); err != nil {
		return err
	}
	return mapPushErr(r.PushContext(ctx, &git.PushOptions{
		Auth:     auth,
		RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:%s", b.refName(), b.refName()))},
	}))
}

func (b GitBackend) firstPush(ctx context.Context, auth transport.AuthMethod, blob []byte) error {
	r, err := git.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		return err
	}
	if _, err := r.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{b.Cfg.URL}}); err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	if err := writeWorktreeFile(wt, b.Cfg.Path, blob); err != nil {
		return err
	}
	if _, err := wt.Add(b.Cfg.Path); err != nil {
		return err
	}
	if _, err := wt.Commit("Create IP space inventory", &git.CommitOptions{Author: gitSignature()}); err != nil {
		return err
	}
	// Push the just-made commit (on the init default branch) to the target branch.
	head, err := r.Head()
	if err != nil {
		return err
	}
	return mapPushErr(r.PushContext(ctx, &git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:%s", head.Name(), b.refName()))},
	}))
}

// Verify connects and confirms the branch is reachable (and so validates the URL
// and credentials). An empty remote is reported so the user makes an initial
// commit first.
func (b GitBackend) Verify(ctx context.Context) error {
	r, err := b.cloneBranch(ctx)
	if err != nil {
		return err
	}
	if r == nil {
		return fmt.Errorf("the repository is empty — make an initial commit (e.g. add a README) before syncing to it")
	}
	return nil
}

// Describe returns the repo/branch label.
func (b GitBackend) Describe() string { return b.Label }

func writeWorktreeFile(wt *git.Worktree, path string, data []byte) error {
	f, err := wt.Filesystem.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func gitSignature() *object.Signature {
	return &object.Signature{Name: "Meridian", Email: "meridian@localhost", When: time.Now()}
}

func mapPushErr(err error) error {
	if err == nil || errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	if errors.Is(err, git.ErrNonFastForwardUpdate) {
		return ErrRemoteChanged
	}
	return err
}
