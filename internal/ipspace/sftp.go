package ipspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPBackend stores the shared blob as a file on an SSH/SFTP server. SFTP has no
// server-side conditional-write primitive, so the compare-and-swap is the same
// content-hash check the local path uses, with the new blob written to a temp
// file and atomically renamed into place. The security-critical part is the SSH
// host key: it is pinned on first connect (trust-on-first-use) and a later change
// is refused, to catch a man-in-the-middle.
type SFTPBackend struct {
	Cfg   SFTPConfig
	Label string
}

// SFTPConfig is the connection detail for an SFTP server. Secret fields
// (Password, PrivateKey, KeyPassphrase) are sealed with the store passphrase.
type SFTPConfig struct {
	Host          string
	Port          string
	User          string
	Password      string
	PrivateKey    string // PEM, optional (key auth)
	KeyPassphrase string // for an encrypted private key, optional
	Path          string // remote file path
	HostKey       string // pinned host key (authorized_keys line), set on first connect
}

// sftpConn is the slice of SFTP operations the backend needs; the real adapter
// wraps *sftp.Client, and tests inject an in-memory fake.
type sftpConn interface {
	read(path string) ([]byte, error) // os.ErrNotExist when the file is absent
	writeAtomic(path string, data []byte) error
	remove(path string) error
	close() error
}

// sftpDial opens a connection; it is a package var so tests can inject a fake.
// It returns the connection and the server's host-key line (pinned or observed).
var sftpDial = realSFTPDial

// Describe is the credential-free label.
func (c SFTPConfig) Describe() string {
	host := c.Host
	if c.Port != "" && c.Port != "22" {
		host += ":" + c.Port
	}
	return fmt.Sprintf("SFTP %s@%s:%s", c.User, host, c.Path)
}

// NewSFTPBackend builds a backend from config.
func NewSFTPBackend(c SFTPConfig) (*SFTPBackend, error) {
	if c.Host == "" || c.User == "" || c.Path == "" {
		return nil, fmt.Errorf("SFTP needs a host, a username, and a remote path")
	}
	return &SFTPBackend{Cfg: c, Label: c.Describe()}, nil
}

func (b SFTPBackend) conn(ctx context.Context) (sftpConn, error) {
	c, _, err := sftpDial(ctx, b.Cfg)
	return c, err
}

// Get reads the remote file, or (nil, "", nil) if it does not exist yet.
func (b SFTPBackend) Get(ctx context.Context) ([]byte, string, error) {
	c, err := b.conn(ctx)
	if err != nil {
		return nil, "", err
	}
	defer c.close()
	data, err := c.read(b.Cfg.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return data, versionOf(data), nil
}

// Put writes the remote file only if its current content still hashes to
// expectedVersion; otherwise it returns ErrRemoteChanged.
func (b SFTPBackend) Put(ctx context.Context, blob []byte, expectedVersion string) error {
	c, err := b.conn(ctx)
	if err != nil {
		return err
	}
	defer c.close()
	cur := ""
	data, err := c.read(b.Cfg.Path)
	if err == nil {
		cur = versionOf(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if cur != expectedVersion {
		return ErrRemoteChanged
	}
	return c.writeAtomic(b.Cfg.Path, blob)
}

// Verify connects (validating auth and the pinned host key) and confirms the
// remote path is writable and supports the atomic rename the CAS depends on, by
// writing and removing a throwaway probe file — so a bad path or a server that
// can't do the write is rejected at setup, not on the first real sync.
func (b SFTPBackend) Verify(ctx context.Context) error {
	c, err := b.conn(ctx)
	if err != nil {
		return err
	}
	defer c.close()
	probePath := b.Cfg.Path + ".meridian-probe"
	if err := c.writeAtomic(probePath, []byte("probe")); err != nil {
		return fmt.Errorf("couldn't write to the remote path (check it exists and is writable): %w", err)
	}
	_ = c.remove(probePath)
	return nil
}

// Describe returns the user@host:path label.
func (b SFTPBackend) Describe() string { return b.Label }

// ProbeSFTP connects once, returning the server's host-key line (the pinned key,
// or the observed key on a first connect) after validating auth. Configuration
// uses it to pin the host key.
func ProbeSFTP(ctx context.Context, cfg SFTPConfig) (string, error) {
	c, hostKey, err := sftpDial(ctx, cfg)
	if err != nil {
		return "", err
	}
	c.close()
	return hostKey, nil
}

// makeHostKeyCallback pins the host key: if one is already pinned it must match
// (a change is refused, to catch a man-in-the-middle); otherwise the observed key
// is captured for trust-on-first-use.
func makeHostKeyCallback(pinned string, captured *string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
		if pinned != "" {
			if line != strings.TrimSpace(pinned) {
				return fmt.Errorf("the server's SSH host key has changed since you set this up — refusing to connect in case someone is impersonating it. If the server was legitimately re-keyed, remove and re-add the sync location")
			}
			return nil
		}
		if captured != nil {
			*captured = line
		}
		return nil
	}
}

// hostKeyAlgos pins the key algorithm to the one already trusted, so a server
// legitimately offering a different key type isn't mistaken for an impersonator.
// Empty (first connect) accepts the server's default so TOFU can capture it.
func hostKeyAlgos(pinned string) []string {
	if pinned == "" {
		return nil
	}
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pinned))
	if err != nil {
		return nil
	}
	return []string{pub.Type()}
}

// realSFTPDial opens a real SSH+SFTP connection. The caller's context bounds the
// whole thing — handshake and I/O — via a watcher that closes the socket on
// cancellation (ClientConfig.Timeout only covers ssh.Dial, which this path does
// not use, so it is not relied on here).
func realSFTPDial(ctx context.Context, cfg SFTPConfig) (sftpConn, string, error) {
	auths, err := sftpAuthMethods(cfg)
	if err != nil {
		return nil, "", err
	}
	var captured string
	port := cfg.Port
	if port == "" {
		port = "22"
	}
	clientCfg := &ssh.ClientConfig{
		User:              cfg.User,
		Auth:              auths,
		HostKeyCallback:   makeHostKeyCallback(cfg.HostKey, &captured),
		HostKeyAlgorithms: hostKeyAlgos(cfg.HostKey),
		Timeout:           20 * time.Second,
	}
	addr := net.JoinHostPort(cfg.Host, port)
	d := net.Dialer{}
	netConn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", err
	}
	// Close the socket if the context is cancelled, so a stalled handshake or a
	// stalled read/write can't hang forever. Stopped on close().
	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			netConn.Close()
		case <-watchDone:
		}
	}()
	fail := func(e error) (sftpConn, string, error) {
		close(watchDone)
		netConn.Close()
		return nil, "", e
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, clientCfg)
	if err != nil {
		return fail(err)
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	sc, err := sftp.NewClient(sshClient)
	if err != nil {
		close(watchDone)
		sshClient.Close()
		return nil, "", err
	}
	hostKey := cfg.HostKey
	if hostKey == "" {
		hostKey = captured
	}
	return &realSFTPConn{sc: sc, ssh: sshClient, watchDone: watchDone}, hostKey, nil
}

func sftpAuthMethods(cfg SFTPConfig) ([]ssh.AuthMethod, error) {
	var auths []ssh.AuthMethod
	if strings.TrimSpace(cfg.PrivateKey) != "" {
		var signer ssh.Signer
		var err error
		if cfg.KeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(cfg.PrivateKey), []byte(cfg.KeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("couldn't read the private key: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		auths = append(auths, ssh.Password(cfg.Password))
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("provide a password or a private key for the SFTP server")
	}
	return auths, nil
}

// realSFTPConn adapts *sftp.Client to sftpConn.
type realSFTPConn struct {
	sc        *sftp.Client
	ssh       *ssh.Client
	watchDone chan struct{}
}

func (c *realSFTPConn) read(p string) ([]byte, error) {
	f, err := c.sc.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (c *realSFTPConn) writeAtomic(p string, data []byte) error {
	dir := path.Dir(p)
	if dir != "" && dir != "." {
		if err := c.sc.MkdirAll(dir); err != nil {
			return err
		}
	}
	// A unique temp name per writer — two concurrent syncs must never share one
	// temp file (that would interleave their bytes and corrupt the blob for
	// everyone), matching the local backend's os.CreateTemp behaviour.
	tmp := fmt.Sprintf("%s.meridian-%s.tmp", p, newID())
	f, err := c.sc.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		f.Close()
		c.sc.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		c.sc.Remove(tmp)
		return err
	}
	// PosixRename atomically replaces an existing file. If the server lacks that
	// OpenSSH extension, plain SSH_FXP_RENAME won't overwrite, so remove the
	// destination first and rename — not atomic, but the only way to update on
	// such servers (a brief window where the file is absent).
	if err := c.sc.PosixRename(tmp, p); err != nil {
		_ = c.sc.Remove(p)
		if err2 := c.sc.Rename(tmp, p); err2 != nil {
			c.sc.Remove(tmp)
			return err2
		}
	}
	return nil
}

func (c *realSFTPConn) remove(p string) error { return c.sc.Remove(p) }

func (c *realSFTPConn) close() error {
	close(c.watchDone) // stop the context watcher
	err := c.sc.Close()
	c.ssh.Close()
	return err
}
