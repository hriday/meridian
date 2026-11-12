package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/certexpiry"
)

// CertExpiryCheck takes a free-form list of hosts (host, host:port, or a URL,
// one per line or comma/space separated), dials each over TLS, and reports how
// soon its certificate expires — soonest first. Direct TLS dials only; no
// third-party service. Hosts that can't be reached are listed with the error.
func (a *App) CertExpiryCheck(input string) []certexpiry.Result {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hosts := certexpiry.SplitHosts(input)
	return certexpiry.Check(ctx, dialLeaf, hosts, time.Now())
}

// dialLeaf opens a TLS connection and returns the leaf certificate the server
// presents. It skips chain/hostname verification on purpose: the job is to read
// the certificate's dates even when it's expired, self-signed, or served for a
// different name — validity is reported separately via the expiry math.
func dialLeaf(ctx context.Context, hostport string) (certexpiry.Leaf, error) {
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
		return certexpiry.Leaf{}, cleanDialErr(err)
	}
	defer conn.Close()
	state := conn.(*tls.Conn).ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return certexpiry.Leaf{}, fmt.Errorf("no certificate presented")
	}
	c := state.PeerCertificates[0]
	return certexpiry.Leaf{
		NotBefore: c.NotBefore,
		NotAfter:  c.NotAfter,
		Subject:   c.Subject.CommonName,
		Issuer:    c.Issuer.CommonName,
		DNSNames:  c.DNSNames,
	}, nil
}

// cleanDialErr trims the noisy address prefixes off common dial failures so the
// UI shows "connection refused" rather than a wall of wrapped errors.
func cleanDialErr(err error) error {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		if tail := strings.TrimSpace(msg[i+2:]); tail != "" {
			msg = tail
		}
	}
	return fmt.Errorf("%s", msg)
}
