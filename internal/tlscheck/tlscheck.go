// Package tlscheck connects to a host over TLS and reports its certificate and
// chain with a plain-language verdict (valid / expired / self-signed /
// untrusted / hostname mismatch / expiring soon / weak crypto). It retrieves
// the presented certificate even when it is untrusted or expired, so a broken
// cert is still inspectable. The engine is UI-free; the CLI and Wails
// controller are thin adapters over Check.
package tlscheck

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// expirySoonDays is how close to expiry (in days) triggers an "Expiring soon"
// warning.
const expirySoonDays = 21

// CertReport summarises a host's TLS certificate and its health.
type CertReport struct {
	Host          string    `json:"host"`
	Port          string    `json:"port"`
	Subject       string    `json:"subject"`
	Issuer        string    `json:"issuer"`
	SANs          []string  `json:"sans"`
	Serial        string    `json:"serial"`
	NotBefore     time.Time `json:"notBefore"`
	NotAfter      time.Time `json:"notAfter"`
	DaysToExpiry  int       `json:"daysToExpiry"`
	KeyType       string    `json:"keyType"`
	KeyBits       int       `json:"keyBits"`
	SigAlg        string    `json:"sigAlg"`
	TLSVersion    string    `json:"tlsVersion"`
	CipherSuite   string    `json:"cipherSuite"`
	SelfSigned    bool      `json:"selfSigned"`
	ChainTrusted  bool      `json:"chainTrusted"`
	HostnameMatch bool      `json:"hostnameMatch"`
	Verdict       string    `json:"verdict"`
	Warnings      []string  `json:"warnings"`
}

// dialTLS opens a TLS connection. It is a package var so tests can point it at a
// local listener.
var dialTLS = func(ctx context.Context, network, addr string, cfg *tls.Config) (*tls.Conn, error) {
	d := tls.Dialer{NetDialer: &net.Dialer{}, Config: cfg}
	c, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	return c.(*tls.Conn), nil
}

// systemRoots returns the roots to verify the chain against. nil means the
// host's system trust store. It is a package var so tests can inject a CA pool.
var systemRoots = func() *x509.CertPool { return nil }

// splitTarget parses host[:port] or a URL into host and port (default 443).
func splitTarget(target string) (host, portStr string) {
	t := strings.TrimSpace(target)
	if i := strings.Index(t, "://"); i >= 0 {
		t = t[i+3:]
	}
	if i := strings.IndexAny(t, "/?#"); i >= 0 {
		t = t[:i]
	}
	if h, p, err := net.SplitHostPort(t); err == nil {
		return h, p
	}
	// No port: a bracketed IPv6 literal ("[::1]") keeps its brackets through the
	// SplitHostPort error path, so strip them before defaulting the port.
	if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		t = t[1 : len(t)-1]
	}
	return t, "443"
}

// Check connects to target (host[:port] or URL, default port 443) and reports
// its certificate. A connect/handshake failure returns an error; a reachable
// host presenting a flawed certificate returns a full report with a non-Valid
// verdict.
func Check(ctx context.Context, target string, timeout time.Duration) (CertReport, error) {
	host, portStr := splitTarget(target)
	if host == "" {
		return CertReport{}, errors.New("no host in target")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := dialTLS(ctx, "tcp", net.JoinHostPort(host, portStr), &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true, // we inspect even untrusted certs
	})
	if err != nil {
		return CertReport{}, err
	}
	defer conn.Close()

	st := conn.ConnectionState()
	if len(st.PeerCertificates) == 0 {
		return CertReport{}, errors.New("server presented no certificate")
	}
	leaf := st.PeerCertificates[0]

	r := CertReport{
		Host:        host,
		Port:        portStr,
		Subject:     leaf.Subject.String(),
		Issuer:      leaf.Issuer.String(),
		SANs:        sans(leaf),
		Serial:      leaf.SerialNumber.String(),
		NotBefore:   leaf.NotBefore,
		NotAfter:    leaf.NotAfter,
		SigAlg:      leaf.SignatureAlgorithm.String(),
		TLSVersion:  tlsVersionName(st.Version),
		CipherSuite: tls.CipherSuiteName(st.CipherSuite),
	}
	r.KeyType, r.KeyBits = keyInfo(leaf)
	r.DaysToExpiry = int(time.Until(leaf.NotAfter).Hours() / 24)
	r.HostnameMatch = leaf.VerifyHostname(host) == nil
	r.SelfSigned = selfSigned(leaf)

	// Trust: verify against the roots plus any presented intermediates. EKU is
	// pinned to server authentication so a cert not meant for TLS servers can't
	// read as fully valid. CurrentTime is left at now for an in-date cert and
	// only backdated for an out-of-date one (see trustTime) — setting it on
	// macOS forces Go off the system verifier onto a stricter fallback that
	// rejects otherwise-valid public chains (e.g. a Google Trust Services leaf)
	// as "not standards compliant".
	inter := x509.NewCertPool()
	for _, c := range st.PeerCertificates[1:] {
		inter.AddCert(c)
	}
	_, verr := leaf.Verify(x509.VerifyOptions{
		Roots:         systemRoots(),
		Intermediates: inter,
		CurrentTime:   trustTime(leaf, time.Now()),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	r.ChainTrusted = verr == nil

	r.Verdict, r.Warnings = assess(&r, leaf)
	return r, nil
}

// trustTime is the time to assess chain trust at. For an in-date cert it returns
// the zero time, which x509.Verify treats as "now" and which keeps macOS on its
// system verifier. Only for an out-of-date cert does it backdate to the leaf's
// NotBefore, so an expired-but-otherwise-trusted cert still reads as trusted
// (its expiry is reported separately); that path is already off the fast system
// verifier, so it costs nothing extra.
func trustTime(leaf *x509.Certificate, now time.Time) time.Time {
	if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
		return leaf.NotBefore
	}
	return time.Time{}
}

// assess builds the warning list and selects the worst-case verdict.
func assess(r *CertReport, leaf *x509.Certificate) (string, []string) {
	now := time.Now()
	type issue struct{ label string }
	var warnings []string
	add := func(s string) { warnings = append(warnings, s) }

	notYet := now.Before(leaf.NotBefore)
	expired := now.After(leaf.NotAfter)
	expiringSoon := !expired && r.DaysToExpiry < expirySoonDays
	weakKey := r.KeyType == "RSA" && r.KeyBits < 2048
	weakSig := weakSignature(leaf.SignatureAlgorithm)

	if notYet {
		add("Not yet valid")
	}
	if expired {
		add("Expired")
	}
	if r.SelfSigned {
		add("Self-signed")
	}
	if !r.ChainTrusted && !r.SelfSigned {
		add("Untrusted chain")
	}
	if !r.HostnameMatch {
		add("Hostname mismatch")
	}
	if expiringSoon {
		add(fmt.Sprintf("Expiring soon (%d days)", r.DaysToExpiry))
	}
	if weakKey {
		add("Weak key")
	}
	if weakSig {
		add("Weak signature")
	}

	// Worst-first verdict.
	order := []issue{
		{"Not yet valid"}, {"Expired"}, {"Self-signed"}, {"Untrusted chain"},
		{"Hostname mismatch"}, {"Expiring soon"}, {"Weak key"}, {"Weak signature"},
	}
	for _, it := range order {
		for _, w := range warnings {
			if strings.HasPrefix(w, it.label) {
				return it.label, warnings
			}
		}
	}
	return "Valid", warnings
}

// selfSigned reports whether a certificate signed itself. It checks the leaf's
// own signature directly rather than via CheckSignatureFrom, which would apply
// CA constraints to the leaf-as-parent and reject the common self-signed server
// cert (CA:FALSE, server KeyUsage).
func selfSigned(c *x509.Certificate) bool {
	if !bytes.Equal(c.RawSubject, c.RawIssuer) {
		return false
	}
	return c.CheckSignature(c.SignatureAlgorithm, c.RawTBSCertificate, c.Signature) == nil
}

func sans(c *x509.Certificate) []string {
	out := make([]string, 0, len(c.DNSNames)+len(c.IPAddresses))
	out = append(out, c.DNSNames...)
	for _, ip := range c.IPAddresses {
		out = append(out, ip.String())
	}
	return out
}

func keyInfo(c *x509.Certificate) (string, int) {
	switch k := c.PublicKey.(type) {
	case *rsa.PublicKey:
		return "RSA", k.N.BitLen()
	case *ecdsa.PublicKey:
		return "ECDSA", k.Curve.Params().BitSize
	case ed25519.PublicKey:
		return "Ed25519", 256
	default:
		return "unknown", 0
	}
}

func weakSignature(a x509.SignatureAlgorithm) bool {
	switch a {
	case x509.MD2WithRSA, x509.MD5WithRSA,
		x509.SHA1WithRSA, x509.DSAWithSHA1, x509.ECDSAWithSHA1:
		return true
	default:
		return false
	}
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}
