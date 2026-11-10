package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/tlscheck"
	"github.com/stretchr/testify/require"
)

func TestPrintTLSReport(t *testing.T) {
	var out bytes.Buffer
	printTLSReport(&out, tlscheck.CertReport{
		Host:         "example.com",
		Port:         "443",
		Subject:      "CN=example.com",
		Issuer:       "CN=Example CA",
		SANs:         []string{"example.com", "www.example.com"},
		NotAfter:     time.Now().Add(10 * 24 * time.Hour),
		DaysToExpiry: 10,
		KeyType:      "RSA",
		KeyBits:      2048,
		SigAlg:       "SHA256-RSA",
		TLSVersion:   "TLS 1.3",
		Verdict:      "Expiring soon (10 days)",
		Warnings:     []string{"Expiring soon (10 days)"},
	})
	s := out.String()
	require.Contains(t, s, "Expiring soon")
	require.Contains(t, s, "CN=example.com")
	require.Contains(t, s, "www.example.com")
	require.Contains(t, s, "RSA")
	require.Contains(t, s, "TLS 1.3")
}

func TestRunTLSMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"tls"}, &out, &errb)
	require.Equal(t, 2, code)
	require.NotEmpty(t, errb.String())
}
