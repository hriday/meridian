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

// Verify connects (validating auth and the pinned host key); SFTP has no
// conditional-write capability to probe beyond that.
func (b SFTPBackend) Verify(ctx context.Context) error {
	c, err := b.conn(ctx)
	if err != nil {
		return err
	}
	return c.close()
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

// realSFTPDial opens a real SSH+SFTP connection.
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
		User:            cfg.User,
		Auth:            auths,
		HostKeyCallback: makeHostKeyCallback(cfg.HostKey, &captured),
		Timeout:         20 * time.Second,
	}
	addr := net.JoinHostPort(cfg.Host, port)
	d := net.Dialer{}
	netConn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, clientCfg)
	if err != nil {
		netConn.Close()
		return nil, "", err
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	sc, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, "", err
	}
	hostKey := cfg.HostKey
	if hostKey == "" {
		hostKey = captured
	}
	return &realSFTPConn{sc: sc, ssh: sshClient}, hostKey, nil
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
	sc  *sftp.Client
	ssh *ssh.Client
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
	tmp := p + ".meridian-tmp"
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
	// PosixRename atomically replaces; fall back to a plain rename if unsupported.
	if err := c.sc.PosixRename(tmp, p); err != nil {
		if rmErr := c.sc.Rename(tmp, p); rmErr != nil {
			c.sc.Remove(tmp)
			return err
		}
	}
	return nil
}

func (c *realSFTPConn) close() error {
	err := c.sc.Close()
	c.ssh.Close()
	return err
}
