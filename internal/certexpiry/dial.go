package certexpiry

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// Dial is the real Probe: it opens a TLS connection and returns the leaf
// certificate the server presents. It skips chain/hostname verification on
// purpose — the job is to read the certificate's dates even when it's expired,
// self-signed, or served for a different name — so validity is reported via the
// expiry math, not the handshake. Shared by the GUI and the CLI.
func Dial(ctx context.Context, hostport string) (Leaf, error) {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	d := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 8 * time.Second},
		Config:    &tls.Config{InsecureSkipVerify: true, ServerName: host}, //nolint:gosec // reading dates, not establishing trust
	}
	conn, err := d.DialContext(ctx, "tcp", hostport)
	if err != nil {
		return Leaf{}, cleanDialErr(err)
	}
	defer conn.Close()
	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return Leaf{}, fmt.Errorf("no certificate presented")
	}
	c := state.PeerCertificates[0]
	return Leaf{
		NotBefore: c.NotBefore,
		NotAfter:  c.NotAfter,
		Subject:   c.Subject.CommonName,
		Issuer:    c.Issuer.CommonName,
		DNSNames:  c.DNSNames,
	}, nil
}

// cleanDialErr trims the noisy address prefixes off common dial failures so the
// output shows "connection refused" rather than a wall of wrapped errors.
func cleanDialErr(err error) error {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		if tail := strings.TrimSpace(msg[i+2:]); tail != "" {
			msg = tail
		}
	}
	return fmt.Errorf("%s", msg)
}
