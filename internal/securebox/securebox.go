// Package securebox seals arbitrary bytes under a passphrase using the same
// scheme as the IP-space store: salt | nonce | secretbox(ciphertext), with the
// key derived from the passphrase by argon2id. It lets sensitive data (e.g. a
// password-audit report) be persisted encrypted at rest.
package securebox

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

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

// Seal encrypts plaintext under passphrase, returning salt|nonce|ciphertext.
func Seal(passphrase string, plaintext []byte) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	var nonce [nonceLen]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	key := deriveKey(passphrase, salt)
	sealed := secretbox.Seal(nil, plaintext, &nonce, &key)

	blob := make([]byte, 0, saltLen+nonceLen+len(sealed))
	blob = append(blob, salt...)
	blob = append(blob, nonce[:]...)
	blob = append(blob, sealed...)
	return blob, nil
}

// Open decrypts a blob produced by Seal. A wrong passphrase or any tamper
// yields an error, never partial plaintext.
func Open(passphrase string, blob []byte) ([]byte, error) {
	if len(blob) < saltLen+nonceLen+secretbox.Overhead {
		return nil, errors.New("securebox: blob is truncated or not a sealed file")
	}
	salt := blob[:saltLen]
	var nonce [nonceLen]byte
	copy(nonce[:], blob[saltLen:saltLen+nonceLen])
	sealed := blob[saltLen+nonceLen:]

	key := deriveKey(passphrase, salt)
	plain, ok := secretbox.Open(nil, sealed, &nonce, &key)
	if !ok {
		return nil, errors.New("couldn't decrypt (wrong passphrase, or the data is corrupt)")
	}
	return plain, nil
}
