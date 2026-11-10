// Package netcat is a raw TCP/UDP connection tool: dial a host:port (optionally
// over TLS) or listen on a local port, exchanging bytes both ways. It never
// executes commands — it only moves bytes.
package netcat

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
)

// Config describes a connection or listen target.
type Config struct {
	Network  string // "tcp" or "udp"
	Addr     string // host:port
	TLS      bool   // wrap the TCP connection in TLS (connect only)
	Insecure bool   // skip TLS verification (probing an invalid/self-signed cert)
}

// CertInfo summarises the peer certificate of a TLS connection.
type CertInfo struct {
	Subject  string `json:"subject"`
	Issuer   string `json:"issuer"`
	NotAfter string `json:"notAfter"`
	Verified bool   `json:"verified"`
}

// Session is one live connection. Start begins a read loop; Send writes; Close
// stops the loop and closes the connection.
type Session struct {
	conn      net.Conn
	closeOnce sync.Once
}

func newSession(c net.Conn) *Session { return &Session{conn: c} }

// Dial opens a connection per cfg. For a TLS connection it returns a CertInfo;
// otherwise CertInfo is nil.
func Dial(ctx context.Context, cfg Config) (*Session, *CertInfo, error) {
	d := net.Dialer{}
	if cfg.TLS {
		if cfg.Network != "tcp" {
			return nil, nil, fmt.Errorf("TLS is only supported over tcp")
		}
		host, _, _ := net.SplitHostPort(cfg.Addr)
		td := tls.Dialer{NetDialer: &d, Config: &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: cfg.Insecure,
		}}
		c, err := td.DialContext(ctx, cfg.Network, cfg.Addr)
		if err != nil {
			return nil, nil, err
		}
		info := certInfo(c.(*tls.Conn), !cfg.Insecure)
		return newSession(c), info, nil
	}
	c, err := d.DialContext(ctx, cfg.Network, cfg.Addr)
	if err != nil {
		return nil, nil, err
	}
	return newSession(c), nil, nil
}

func certInfo(c *tls.Conn, verified bool) *CertInfo {
	st := c.ConnectionState()
	if len(st.PeerCertificates) == 0 {
		return &CertInfo{Verified: verified}
	}
	cert := st.PeerCertificates[0]
	return &CertInfo{
		Subject:  cert.Subject.String(),
		Issuer:   cert.Issuer.String(),
		NotAfter: cert.NotAfter.Format("2006-01-02"),
		Verified: verified,
	}
}

// Send writes b to the connection.
func (s *Session) Send(b []byte) (int, error) { return s.conn.Write(b) }

// Peer returns the remote address.
func (s *Session) Peer() string { return s.conn.RemoteAddr().String() }

// Start runs the read loop in a goroutine, delivering each chunk to onData and
// calling onClose exactly once when the connection ends.
func (s *Session) Start(onData func([]byte), onClose func(error)) {
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := s.conn.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				onData(chunk)
			}
			if err != nil {
				if onClose != nil {
					onClose(err)
				}
				return
			}
		}
	}()
}

// Close closes the connection (idempotent), which also ends the read loop.
func (s *Session) Close() error {
	var err error
	s.closeOnce.Do(func() { err = s.conn.Close() })
	return err
}

// HexDump renders b as an offset/hex/ascii dump (16 bytes per line).
func HexDump(b []byte) string {
	var sb strings.Builder
	for off := 0; off < len(b); off += 16 {
		end := off + 16
		if end > len(b) {
			end = len(b)
		}
		row := b[off:end]
		fmt.Fprintf(&sb, "%08x  ", off)
		for i := 0; i < 16; i++ {
			if i < len(row) {
				fmt.Fprintf(&sb, "%02x ", row[i])
			} else {
				sb.WriteString("   ")
			}
		}
		sb.WriteString(" ")
		for _, c := range row {
			if c >= 0x20 && c < 0x7f {
				sb.WriteByte(c)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
