package ipspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrRemoteChanged means the shared blob moved since it was last pulled: another
// person pushed in between. The caller must pull and review again rather than
// overwrite — this is the compare-and-swap that stops one sync silently
// clobbering another's push.
var ErrRemoteChanged = errors.New("the shared copy changed since you last pulled — pull & review again")

// A Backend stores and retrieves the single encrypted event-log blob at a
// location collaborators share (a mounted path today; object storage, SFTP, and
// git in later slices). Meridian is always the client: a backend only Gets and
// Puts an opaque blob and never runs a service for anyone else. The blob is
// sealed and opened locally (SealEvents/OpenEvents), so a backend never sees the
// passphrase or the plaintext.
//
// Get returns a version token identifying the blob it read; Put takes the
// version the caller expects the remote to still hold and must refuse with
// ErrRemoteChanged if the remote has moved on, so concurrent syncs can't lose a
// write. Later backends map the token to a native primitive (an S3 ETag, a git
// ref); the local backend uses the blob's content hash.
type Backend interface {
	// Get returns the stored blob and its version, or (nil, "", nil) when
	// nothing is there yet (a remote that has never been pushed to).
	Get(ctx context.Context) (blob []byte, version string, err error)
	// Put replaces the stored blob only if the remote's current version still
	// equals expectedVersion ("" meaning "I expect no remote yet"); otherwise it
	// returns ErrRemoteChanged and leaves the remote untouched.
	Put(ctx context.Context, blob []byte, expectedVersion string) error
	// Verify confirms, once at configure time, that this backend can sync safely
	// — in particular that it honors the conditional writes the compare-and-swap
	// relies on. It may perform a cheap round-trip (and so validates reachability
	// and credentials too). A filesystem path always qualifies.
	Verify(ctx context.Context) error
	// Describe is a short, human-readable location for the UI.
	Describe() string
}

// LocalBackend reads and writes the shared blob at a filesystem path — a mounted
// NAS share, a synced folder (Drive/Dropbox/iCloud), or a USB drive. The simplest
// backend, and the one that covers "or wherever". Its version token is the
// content hash of the blob, so a Put is refused whenever the file on disk differs
// from what the caller pulled. On a single filesystem this is effectively a
// compare-and-swap; across an eventually-consistent synced folder it still
// catches every concurrent write that has already propagated, and the merge base
// kept locally means a refused push is re-offered on the next sync rather than
// lost.
type LocalBackend struct {
	Path string
}

func versionOf(blob []byte) string {
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

// Get returns the blob at the path and its content-hash version, or
// (nil, "", nil) if the file does not exist yet.
func (b LocalBackend) Get(ctx context.Context) ([]byte, string, error) {
	blob, err := os.ReadFile(b.Path)
	if os.IsNotExist(err) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return blob, versionOf(blob), nil
}

// Put writes the blob atomically (temp file + rename), but only after confirming
// the remote still holds expectedVersion; otherwise it returns ErrRemoteChanged.
func (b LocalBackend) Put(ctx context.Context, blob []byte, expectedVersion string) error {
	// Re-read the current remote and compare against what the caller pulled.
	_, curVer, err := b.Get(ctx)
	if err != nil {
		return err
	}
	if curVer != expectedVersion {
		return ErrRemoteChanged
	}

	dir := filepath.Dir(b.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ipspace-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(blob); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, b.Path)
}

// Verify is a no-op for a filesystem path: its CAS is the content-hash check in
// Put, which always works, so there is nothing to probe.
func (b LocalBackend) Verify(ctx context.Context) error { return nil }

// Describe returns the configured path.
func (b LocalBackend) Describe() string {
	return fmt.Sprintf("local path %s", b.Path)
}

// Pull fetches the remote blob through the backend and decrypts it, returning
// the records and the version to pass back to Push. It returns (nil, "", nil)
// when the remote has no store yet (a first sync into an empty remote).
func Pull(ctx context.Context, b Backend, passphrase string) ([]Event, string, error) {
	blob, ver, err := b.Get(ctx)
	if err != nil {
		return nil, "", err
	}
	if blob == nil {
		return nil, "", nil
	}
	evs, err := OpenEvents(passphrase, blob)
	if err != nil {
		return nil, "", err
	}
	return evs, ver, nil
}

// Push encrypts the log and writes it through the backend, refusing (with
// ErrRemoteChanged) if the remote moved past expectedVersion since it was pulled.
func Push(ctx context.Context, b Backend, passphrase string, events []Event, expectedVersion string) error {
	blob, err := SealEvents(passphrase, events)
	if err != nil {
		return err
	}
	return b.Put(ctx, blob, expectedVersion)
}
