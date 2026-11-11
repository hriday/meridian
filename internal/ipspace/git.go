package ipspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
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

// GitConfig is the connection detail for a git remote over HTTPS. The secret
// field (Token) is sealed with the store passphrase. HTTPS-token auth only: it
// works everywhere (GitHub/GitLab/Bitbucket/self-hosted) and avoids the SSH
// known_hosts/host-key management an SSH path would need.
type GitConfig struct {
	URL      string // https://…
	Branch   string // default "main"
	Path     string // file path within the repo, e.g. "ipspace.enc"
	Username string // for HTTPS token auth (default "git")
	Token    string // HTTPS personal access token / password
}

func (c GitConfig) branch() string {
	if strings.TrimSpace(c.Branch) == "" {
		return "main"
	}
	return strings.TrimSpace(c.Branch)
}

// Describe is the credential-free label. Any userinfo embedded in the URL (e.g.
// a pasted https://user:token@host) is stripped so a token can't leak into the
// UI or an error string.
func (c GitConfig) Describe() string {
	return fmt.Sprintf("git %s (%s) [%s]", sanitizeURL(c.URL), c.branch(), c.Path)
}

// sanitizeURL removes any embedded credentials (userinfo) from a URL.
func sanitizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		u.User = nil
	}
	return u.String()
}

func (c GitConfig) auth() transport.AuthMethod {
	if c.Token != "" {
		user := c.Username
		if user == "" {
			user = "git"
		}
		return &githttp.BasicAuth{Username: user, Password: c.Token}
	}
	return nil // local transport / no auth
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
	r, err := git.CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           b.Cfg.URL,
		Auth:          b.Cfg.auth(),
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
		return b.firstPush(ctx, b.Cfg.auth(), blob)
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	if err := b.stageBlob(wt, blob); err != nil {
		return err
	}
	if _, err := wt.Commit("Update IP space inventory", &git.CommitOptions{Author: gitSignature()}); err != nil {
		return err
	}
	return mapPushErr(r.PushContext(ctx, &git.PushOptions{
		Auth:     b.Cfg.auth(),
		RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("%s:%s", b.refName(), b.refName()))},
	}))
}

// stageBlob writes the blob and a .gitattributes entry marking it binary (so a
// human later using real git on the repo can't corrupt the sealed bytes via
// text/eol normalization), then stages both.
func (b GitBackend) stageBlob(wt *git.Worktree, blob []byte) error {
	if err := writeWorktreeFile(wt, b.Cfg.Path, blob); err != nil {
		return err
	}
	if _, err := wt.Add(b.Cfg.Path); err != nil {
		return err
	}
	if err := ensureGitattributes(wt, b.Cfg.Path); err != nil {
		return err
	}
	return nil
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
	if err := b.stageBlob(wt, blob); err != nil {
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

// ensureGitattributes adds a line marking the blob path binary, if not already
// present, and stages the file.
func ensureGitattributes(wt *git.Worktree, blobPath string) error {
	const name = ".gitattributes"
	marker := blobPath + " -text"
	existing := ""
	if f, err := wt.Filesystem.Open(name); err == nil {
		b, _ := io.ReadAll(f)
		f.Close()
		existing = string(b)
	}
	if strings.Contains(existing, marker) {
		return nil
	}
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	if err := writeWorktreeFile(wt, name, []byte(existing+marker+"\n")); err != nil {
		return err
	}
	_, err := wt.Add(name)
	return err
}

func gitSignature() *object.Signature {
	return &object.Signature{Name: "Meridian", Email: "meridian@localhost", When: time.Now()}
}

func mapPushErr(err error) error {
	if err == nil || errors.Is(err, git.NoErrAlreadyUpToDate) {
		return nil
	}
	// A rejected push is a concurrent-write race. go-git does NOT return the
	// git.ErrNonFastForwardUpdate sentinel for a *push* (that's the pull path);
	// the push error is a plain message, and a real server's report-status
	// rejection is another phrasing — so match the sentinel AND the known
	// messages, mapping all of them to the CAS "remote changed" signal.
	if errors.Is(err, git.ErrNonFastForwardUpdate) || isNonFastForward(err) {
		return ErrRemoteChanged
	}
	return err
}

func isNonFastForward(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "non-fast-forward") ||
		strings.Contains(s, "fetch first") ||
		strings.Contains(s, "stale info") ||
		strings.Contains(s, "cannot lock ref")
}
