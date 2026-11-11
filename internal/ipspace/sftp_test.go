package ipspace

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// memSFTP is an in-memory sftpConn for testing the backend's CAS/version logic
// without a network or an SSH server.
type memSFTP struct {
	files  map[string][]byte
	closed bool
}

func newMemSFTP() *memSFTP { return &memSFTP{files: map[string][]byte{}} }
func (m *memSFTP) read(path string) ([]byte, error) {
	b, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return b, nil
}
func (m *memSFTP) writeAtomic(path string, data []byte) error {
	m.files[path] = append([]byte(nil), data...)
	return nil
}
func (m *memSFTP) remove(path string) error { delete(m.files, path); return nil }
func (m *memSFTP) close() error             { m.closed = true; return nil }

// withFakeDial swaps sftpDial for the duration of a test.
func withFakeDial(t *testing.T, conn sftpConn, hostKey string, dialErr error) {
	t.Helper()
	orig := sftpDial
	sftpDial = func(ctx context.Context, cfg SFTPConfig) (sftpConn, string, error) {
		if dialErr != nil {
			return nil, "", dialErr
		}
		return conn, hostKey, nil
	}
	t.Cleanup(func() { sftpDial = orig })
}

func sftpB() SFTPBackend {
	return SFTPBackend{Cfg: SFTPConfig{Host: "h", User: "u", Path: "/srv/ipspace.enc"}, Label: "SFTP u@h:/srv/ipspace.enc"}
}

func TestSFTPGetMissingIsNil(t *testing.T) {
	withFakeDial(t, newMemSFTP(), "hk", nil)
	blob, ver, err := sftpB().Get(context.Background())
	require.NoError(t, err)
	require.Nil(t, blob)
	require.Empty(t, ver)
}

func TestSFTPPutGetRoundTrip(t *testing.T) {
	withFakeDial(t, newMemSFTP(), "hk", nil)
	b := sftpB()
	require.NoError(t, b.Put(context.Background(), []byte("hello"), ""))
	blob, ver, err := b.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), blob)
	require.NotEmpty(t, ver)
}

func TestSFTPPutStaleIsRemoteChanged(t *testing.T) {
	withFakeDial(t, newMemSFTP(), "hk", nil)
	b := sftpB()
	require.NoError(t, b.Put(context.Background(), []byte("v1"), ""))
	_, v1, _ := b.Get(context.Background())
	require.NoError(t, b.Put(context.Background(), []byte("v2"), v1)) // moves it on
	err := b.Put(context.Background(), []byte("v1-edit"), v1)         // stale
	require.ErrorIs(t, err, ErrRemoteChanged)
}

func TestSFTPVerifyWritesAndCleansUp(t *testing.T) {
	m := newMemSFTP()
	withFakeDial(t, m, "hk", nil)
	require.NoError(t, sftpB().Verify(context.Background()))
	require.True(t, m.closed, "Verify closes the connection it opened")
	require.Empty(t, m.files, "Verify removes its probe file, leaving the path clean")
}

func TestSFTPVerifyFailsOnUnwritablePath(t *testing.T) {
	withFakeDial(t, &writeFailSFTP{}, "hk", nil)
	require.Error(t, sftpB().Verify(context.Background()))
}

func TestHostKeyAlgosPinsToKeyType(t *testing.T) {
	key := testPubKey(t)
	line := string(ssh.MarshalAuthorizedKey(key))
	require.Equal(t, []string{key.Type()}, hostKeyAlgos(line))
	require.Nil(t, hostKeyAlgos(""), "no pin on first connect, so TOFU can capture any type")
}

// writeFailSFTP is a conn whose writes fail, to test Verify's writability check.
type writeFailSFTP struct{ memSFTP }

func (w *writeFailSFTP) writeAtomic(path string, data []byte) error { return os.ErrPermission }

func TestProbeSFTPReturnsHostKey(t *testing.T) {
	withFakeDial(t, newMemSFTP(), "ssh-ed25519 AAAAhostkey", nil)
	hk, err := ProbeSFTP(context.Background(), SFTPConfig{Host: "h", User: "u", Path: "/x"})
	require.NoError(t, err)
	require.Equal(t, "ssh-ed25519 AAAAhostkey", hk)
}

func testPubKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	k, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	return k
}

func TestHostKeyCallbackCapturesWhenUnpinned(t *testing.T) {
	var captured string
	cb := makeHostKeyCallback("", &captured)
	require.NoError(t, cb("h:22", nil, testPubKey(t)))
	require.NotEmpty(t, captured, "an unpinned callback captures the host key for TOFU")
}

func TestHostKeyCallbackAcceptsMatch(t *testing.T) {
	key := testPubKey(t)
	line := string(ssh.MarshalAuthorizedKey(key))
	cb := makeHostKeyCallback(line, nil)
	require.NoError(t, cb("h:22", nil, key))
}

func TestHostKeyCallbackRejectsMismatch(t *testing.T) {
	cb := makeHostKeyCallback("ssh-ed25519 AAAApinnedDIFFERENT", nil)
	err := cb("h:22", nil, testPubKey(t))
	require.Error(t, err, "a changed host key must be refused (MITM protection)")
}
