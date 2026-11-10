package tlscheck

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSplitTarget(t *testing.T) {
	cases := map[string][2]string{
		"example.com":           {"example.com", "443"},
		"example.com:8443":      {"example.com", "8443"},
		"https://example.com/x": {"example.com", "443"},
		"http://h.test:9000/a":  {"h.test", "9000"},
		"[::1]:8443":            {"::1", "8443"},
		"[::1]":                 {"::1", "443"},
		"::1":                   {"::1", "443"},
	}
	for in, want := range cases {
		h, p := splitTarget(in)
		require.Equal(t, want[0], h, in)
		require.Equal(t, want[1], p, in)
	}
}

// ---- test cert/listener helpers ----

type leafSpec struct {
	sans      []string
	notBefore time.Time
	notAfter  time.Time
	bits      int
}

func defaultLeaf() leafSpec {
	return leafSpec{
		sans:      []string{"localhost"},
		notBefore: time.Now().Add(-time.Hour),
		notAfter:  time.Now().Add(365 * 24 * time.Hour),
		bits:      2048,
	}
}

// serveLeaf builds a CA + a leaf per spec, serves the leaf on a one-shot TLS
// listener, and points systemRoots at the CA pool. Returns the listener addr.
// If trust is false, systemRoots is set to an empty pool (untrusted).
func serveLeaf(t *testing.T, spec leafSpec, trust bool) string {
	t.Helper()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Meridian Test CA"},
		NotBefore:             time.Now().Add(-10 * 365 * 24 * time.Hour), // long-lived CA, like real ones
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	leafKey, err := rsa.GenerateKey(rand.Reader, spec.bits)
	require.NoError(t, err)
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: spec.sans[0]},
		NotBefore:    spec.notBefore,
		NotAfter:     spec.notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     spec.sans,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	tlsCert := tls.Certificate{Certificate: [][]byte{leafDER, caDER}, PrivateKey: leafKey}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{tlsCert}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			if tc, ok := c.(*tls.Conn); ok {
				_ = tc.Handshake()
			}
			_ = c.Close()
		}
	}()

	pool := x509.NewCertPool()
	if trust {
		pool.AddCert(caCert)
	}
	old := systemRoots
	systemRoots = func() *x509.CertPool { return pool }
	t.Cleanup(func() { systemRoots = old })
	return ln.Addr().String()
}

// serveSelfSigned serves a genuine self-signed server leaf (CA:FALSE, server
// KeyUsage, subject==issuer) on a one-shot TLS listener with an empty roots
// pool, and returns the addr.
func serveSelfSigned(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(7),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			if tc, ok := c.(*tls.Conn); ok {
				_ = tc.Handshake()
			}
			_ = c.Close()
		}
	}()
	old := systemRoots
	systemRoots = func() *x509.CertPool { return x509.NewCertPool() }
	t.Cleanup(func() { systemRoots = old })
	return ln.Addr().String()
}

func port(addr string) string {
	_, p, _ := net.SplitHostPort(addr)
	return p
}

func TestCheckTrustedValid(t *testing.T) {
	addr := serveLeaf(t, defaultLeaf(), true)
	r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "Valid", r.Verdict)
	require.True(t, r.ChainTrusted)
	require.True(t, r.HostnameMatch)
	require.False(t, r.SelfSigned)
	require.Equal(t, "RSA", r.KeyType)
	require.Equal(t, 2048, r.KeyBits)
	require.Contains(t, r.SANs, "localhost")
	require.Greater(t, r.DaysToExpiry, 0)
	require.Empty(t, r.Warnings)
}

func TestCheckExpired(t *testing.T) {
	spec := defaultLeaf()
	spec.notBefore = time.Now().Add(-48 * time.Hour)
	spec.notAfter = time.Now().Add(-24 * time.Hour)
	addr := serveLeaf(t, spec, true)
	r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "Expired", r.Verdict)
	require.Contains(t, r.Warnings, "Expired")
	require.True(t, r.ChainTrusted) // trust is independent of expiry
}

func TestCheckUntrustedChain(t *testing.T) {
	// A CA-signed leaf whose CA is NOT in the roots pool: not self-signed, but
	// the chain doesn't reach a trusted root.
	addr := serveLeaf(t, defaultLeaf(), false)
	r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
	require.NoError(t, err)
	require.False(t, r.ChainTrusted)
	require.False(t, r.SelfSigned)
	require.Equal(t, "Untrusted chain", r.Verdict)
	require.Contains(t, r.Warnings, "Untrusted chain")
}

func TestCheckSelfSigned(t *testing.T) {
	// A genuine self-signed server leaf: CA:FALSE, server KeyUsage, its own issuer.
	addr := serveSelfSigned(t)
	r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
	require.NoError(t, err)
	require.True(t, r.SelfSigned)
	require.False(t, r.ChainTrusted)
	require.Equal(t, "Self-signed", r.Verdict)
	require.Contains(t, r.Warnings, "Self-signed")
}

func TestCheckHostnameMismatch(t *testing.T) {
	spec := defaultLeaf()
	spec.sans = []string{"example.com"}
	addr := serveLeaf(t, spec, true)
	r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
	require.NoError(t, err)
	require.False(t, r.HostnameMatch)
	require.True(t, r.ChainTrusted)
	require.Equal(t, "Hostname mismatch", r.Verdict)
}

func TestCheckWeakKey(t *testing.T) {
	spec := defaultLeaf()
	spec.bits = 1024
	addr := serveLeaf(t, spec, true)
	r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, 1024, r.KeyBits)
	require.Equal(t, "Weak key", r.Verdict)
	require.Contains(t, r.Warnings, "Weak key")
}

func TestCheckDialError(t *testing.T) {
	_, err := Check(context.Background(), "127.0.0.1:1", 1*time.Second)
	require.Error(t, err)
}
