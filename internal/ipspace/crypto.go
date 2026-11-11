package ipspace

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

// On-disk format: salt | nonce | secretbox(ciphertext). The key is derived from
// the passphrase with argon2id over the stored salt.
const (
	saltLen  = 16
	nonceLen = 24
	keyLen   = 32

	argonTime    = 1
	argonMem     = 64 * 1024 // 64 MiB
	argonThreads = 4
)

func deriveKey(passphrase string, salt []byte) [keyLen]byte {
	k := argon2.IDKey([]byte(passphrase), salt, argonTime, argonMem, argonThreads, keyLen)
	var key [keyLen]byte
	copy(key[:], k)
	return key
}

// Seal encrypts arbitrary plaintext into a self-contained blob
// (salt | nonce | secretbox) with a key derived from the passphrase. This is the
// format Save writes on disk and the sync backends carry, and it also seals the
// sync config (which may hold backend credentials).
func Seal(passphrase string, plain []byte) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	var nonce [nonceLen]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	key := deriveKey(passphrase, salt)
	sealed := secretbox.Seal(nil, plain, &nonce, &key)

	blob := make([]byte, 0, saltLen+nonceLen+len(sealed))
	blob = append(blob, salt...)
	blob = append(blob, nonce[:]...)
	blob = append(blob, sealed...)
	return blob, nil
}

// Open decrypts a blob produced by Seal. A wrong passphrase or any tamper yields
// a clear error rather than garbage data.
func Open(passphrase string, blob []byte) ([]byte, error) {
	if len(blob) < saltLen+nonceLen+secretbox.Overhead {
		return nil, errors.New("ipspace: store blob is truncated or not a Meridian store")
	}
	salt := blob[:saltLen]
	var nonce [nonceLen]byte
	copy(nonce[:], blob[saltLen:saltLen+nonceLen])
	sealed := blob[saltLen+nonceLen:]

	key := deriveKey(passphrase, salt)
	plain, ok := secretbox.Open(nil, sealed, &nonce, &key)
	if !ok {
		return nil, errors.New("couldn't unlock the IP-space store (wrong passphrase, or the file is corrupt)")
	}
	return plain, nil
}

// SealEvents encrypts the event log into the on-disk / wire blob format.
func SealEvents(passphrase string, events []Event) ([]byte, error) {
	plain, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}
	return Seal(passphrase, plain)
}

// OpenEvents decrypts and parses a blob produced by SealEvents.
func OpenEvents(passphrase string, blob []byte) ([]Event, error) {
	plain, err := Open(passphrase, blob)
	if err != nil {
		return nil, err
	}
	var events []Event
	if err := json.Unmarshal(plain, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// Save writes the event log to path as a passphrase-encrypted blob, atomically
// (temp file + rename).
func Save(path, passphrase string, events []Event) error {
	blob, err := SealEvents(passphrase, events)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ipspace-*")
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
	return os.Rename(tmpName, path)
}

// Load decrypts and parses the event log. A wrong passphrase or any tamper
// yields a clear error rather than partial/garbage data.
func Load(path, passphrase string) ([]Event, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return OpenEvents(passphrase, blob)
}

// Exists reports whether a store file is present at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Create writes a new, empty encrypted store.
func Create(path, passphrase string) error {
	return Save(path, passphrase, []Event{})
}
