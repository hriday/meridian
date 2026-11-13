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
// accepted suite.
func fakeServer(versions map[uint16]bool, acceptCipher uint16) Handshaker {
	return func(ctx context.Context, hostport string, cfg *tls.Config) (tls.ConnectionState, error) {
		v := cfg.MaxVersion // Scan pins Min==Max
		if !versions[v] {
			return tls.ConnectionState{}, errors.New("protocol not supported")
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
	}, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)

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
	}, tls.TLS_RSA_WITH_RC4_128_SHA)

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
