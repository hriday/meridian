package ipspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// A Backend stores and retrieves the single encrypted event-log blob at a
// location collaborators share (a mounted path today; object storage, SFTP, and
// git in later slices). Meridian is always the client: a backend only Gets and
// Puts an opaque blob and never runs a service for anyone else. The blob is
// sealed and opened locally (SealEvents/OpenEvents), so a backend never sees the
// passphrase or the plaintext.
type Backend interface {
	// Get returns the stored blob, or (nil, nil) when nothing is there yet
	// (a remote that has never been pushed to).
	Get(ctx context.Context) ([]byte, error)
	// Put replaces the stored blob.
	Put(ctx context.Context, blob []byte) error
	// Describe is a short, human-readable location for the UI.
	Describe() string
}

// LocalBackend reads and writes the shared blob at a filesystem path — a mounted
// NAS share, a synced folder (Drive/Dropbox/iCloud), or a USB drive. The simplest
// backend, and the one that covers "or wherever".
type LocalBackend struct {
	Path string
}

// Get returns the blob at the path, or (nil, nil) if the file does not exist yet.
func (b LocalBackend) Get(ctx context.Context) ([]byte, error) {
	blob, err := os.ReadFile(b.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return blob, nil
}

// Put writes the blob atomically (temp file + rename) so a reader never sees a
// half-written store.
func (b LocalBackend) Put(ctx context.Context, blob []byte) error {
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

// Describe returns the configured path.
func (b LocalBackend) Describe() string {
	return fmt.Sprintf("local path %s", b.Path)
}

// Pull fetches the remote blob through the backend and decrypts it. It returns
// (nil, nil) when the remote has no store yet (a first sync into an empty remote).
func Pull(ctx context.Context, b Backend, passphrase string) ([]Event, error) {
	blob, err := b.Get(ctx)
	if err != nil {
		return nil, err
	}
	if blob == nil {
		return nil, nil
	}
	return OpenEvents(passphrase, blob)
}

// Push encrypts the log and writes it through the backend.
func Push(ctx context.Context, b Backend, passphrase string, events []Event) error {
	blob, err := SealEvents(passphrase, events)
	if err != nil {
		return err
	}
	return b.Put(ctx, blob)
}
