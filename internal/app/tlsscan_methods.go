package app

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/hriday/meridian/internal/tlsscan"
)

// TLSScan probes which TLS versions and cipher suites a host accepts and grades
// the handshake, flagging legacy protocols and weak ciphers. Direct handshakes
// only; no third-party service. It opens several short connections (one per
// protocol and per probed cipher), so it's heavier than a single TLS check.
func (a *App) TLSScan(host string) (tlsscan.Report, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	return tlsscan.Scan(ctx, tlsHandshake, host)
}

// tlsHandshake dials hostport with cfg and returns the negotiated state. It
// never validates the certificate (cfg already sets InsecureSkipVerify) — the
// scan is about the handshake parameters, not trust.
func tlsHandshake(ctx context.Context, hostport string, cfg *tls.Config) (tls.ConnectionState, error) {
	d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 6 * time.Second}, Config: cfg}
	conn, err := d.DialContext(ctx, "tcp", hostport)
	if err != nil {
		return tls.ConnectionState{}, err
	}
	defer conn.Close()
	return conn.(*tls.Conn).ConnectionState(), nil
}
