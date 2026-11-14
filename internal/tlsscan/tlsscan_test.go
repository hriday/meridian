package tlsscan

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeServer accepts a handshake when the single negotiated version is in
// `versions` and (for <=1.2) the offered cipher list is empty or contains an
// accepted suite. If the client pins a post-quantum curve, it succeeds only
// when pqOK is true — modelling a server that does or doesn't support PQ kex.
func fakeServer(versions map[uint16]bool, acceptCipher uint16, pqOK bool) Handshaker {
	return func(ctx context.Context, hostport string, cfg *tls.Config) (tls.ConnectionState, error) {
		v := cfg.MaxVersion // Scan pins Min==Max
		if !versions[v] {
			return tls.ConnectionState{}, errors.New("protocol not supported")
		}
		for _, c := range cfg.CurvePreferences {
			if c == tls.X25519MLKEM768 && !pqOK {
				return tls.ConnectionState{}, errors.New("no mutual key exchange group")
			}
		}
		if v <= tls.VersionTLS12 && len(cfg.CipherSuites) > 0 {
			ok := false
			for _, c := range cfg.CipherSuites {
				if c == acceptCipher {
					ok = true
				}
			}
			if !ok {
				return tls.ConnectionState{}, errors.New("no cipher")
			}
			return tls.ConnectionState{Version: v, CipherSuite: acceptCipher}, nil
		}
		return tls.ConnectionState{Version: v, CipherSuite: tls.TLS_AES_128_GCM_SHA256}, nil
	}
}

func TestScanModernServer(t *testing.T) {
	hs := fakeServer(map[uint16]bool{
		tls.VersionTLS12: true, tls.VersionTLS13: true,
	}, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, true)

	r, err := Scan(context.Background(), hs, "example.com")
	require.NoError(t, err)

	got := map[string]bool{}
	for _, p := range r.Protocols {
		got[p.Name] = p.Supported
	}
	require.False(t, got["TLS 1.0"])
	require.False(t, got["TLS 1.1"])
	require.True(t, got["TLS 1.2"])
	require.True(t, got["TLS 1.3"])

	// The accepted 1.2 suite shows up and is marked secure.
	var sawSecure bool
	for _, c := range r.Ciphers {
		if c.ID == tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 {
			sawSecure = c.Secure
		}
	}
	require.True(t, sawSecure, "ECDHE-RSA-AES128-GCM should be reported secure")
	require.Contains(t, []string{"A", "B"}, r.Grade, "modern config grades high")
}

func TestScanLegacyServerFailsGrade(t *testing.T) {
	// Supports TLS 1.0 and accepts RC4 — both are findings.
	hs := fakeServer(map[uint16]bool{
		tls.VersionTLS10: true, tls.VersionTLS12: true,
	}, tls.TLS_RSA_WITH_RC4_128_SHA, false)

	r, err := Scan(context.Background(), hs, "old.example:443")
	require.NoError(t, err)
	require.Equal(t, "F", r.Grade)

	joined := ""
	for _, f := range r.Findings {
		joined += f + "\n"
	}
	require.Contains(t, joined, "TLS 1.0")
	require.Contains(t, joined, "RC4")
}

func TestScanDetectsPostQuantum(t *testing.T) {
	hs := fakeServer(map[uint16]bool{tls.VersionTLS13: true},
		tls.TLS_AES_128_GCM_SHA256, true)
	r, err := Scan(context.Background(), hs, "pq.example.com")
	require.NoError(t, err)
	require.True(t, r.PQ.Supported, "PQ-capable server should be detected")
	require.Equal(t, "X25519MLKEM768", r.PQ.Group)
}

func TestScanNoPostQuantumIsFlagged(t *testing.T) {
	hs := fakeServer(map[uint16]bool{tls.VersionTLS13: true},
		tls.TLS_AES_128_GCM_SHA256, false)
	r, err := Scan(context.Background(), hs, "classic.example.com")
	require.NoError(t, err)
	require.False(t, r.PQ.Supported)
	joined := ""
	for _, f := range r.Findings {
		joined += f + "\n"
	}
	require.Contains(t, joined, "harvest-now-decrypt-later")
	// PQ absence must NOT fail the letter grade — TLS PQ is still rolling out.
	require.Contains(t, []string{"A", "B"}, r.Grade)
}

func TestScanLegacyCipherCapsAtB(t *testing.T) {
	// A CBC suite (insecure per Go, but not broken) negotiable alongside a
	// modern TLS 1.3 handshake should cap at B, not fail to F.
	hs := fakeServer(map[uint16]bool{
		tls.VersionTLS12: true, tls.VersionTLS13: true,
	}, tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, false)
	r, err := Scan(context.Background(), hs, "legacy-cipher.example")
	require.NoError(t, err)
	require.Equal(t, "B", r.Grade)
}

func TestScanLegacyProtocolCapsAtC(t *testing.T) {
	// TLS 1.0 enabled (with a strong cipher, no broken suites) caps at C.
	hs := fakeServer(map[uint16]bool{
		tls.VersionTLS10: true, tls.VersionTLS12: true,
	}, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, false)
	r, err := Scan(context.Background(), hs, "legacy-proto.example")
	require.NoError(t, err)
	require.Equal(t, "C", r.Grade)
}

func TestScanBrokenCipherIsF(t *testing.T) {
	// RC4 is broken — a hard F even on TLS 1.2.
	hs := fakeServer(map[uint16]bool{tls.VersionTLS12: true},
		tls.TLS_RSA_WITH_RC4_128_SHA, false)
	r, err := Scan(context.Background(), hs, "rc4.example")
	require.NoError(t, err)
	require.Equal(t, "F", r.Grade)
}

func TestScanNotATLSServer(t *testing.T) {
	hs := func(ctx context.Context, hostport string, cfg *tls.Config) (tls.ConnectionState, error) {
		return tls.ConnectionState{}, errors.New("connection refused")
	}
	_, err := Scan(context.Background(), hs, "nothing.example")
	require.Error(t, err)
}

func TestNormalizeHostPort(t *testing.T) {
	require.Equal(t, "example.com:443", normalizeHostPort("example.com"))
	require.Equal(t, "example.com:8443", normalizeHostPort("example.com:8443"))
	require.Equal(t, "example.com:443", normalizeHostPort("https://example.com/x"))
}
