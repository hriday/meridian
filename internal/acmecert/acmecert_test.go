package acmecert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChallengeName(t *testing.T) {
	require.Equal(t, "_acme-challenge.example.com", challengeName("example.com"))
	require.Equal(t, "_acme-challenge.www.example.com", challengeName("www.example.com"))
	require.Equal(t, "_acme-challenge.example.com", challengeName("*.example.com"), "wildcard validates against the base name")
	require.Equal(t, "_acme-challenge.example.com", challengeName("  example.com  "))
}

func TestCleanDomains(t *testing.T) {
	got := cleanDomains([]string{"Example.com", "example.com.", "  www.example.com ", "", "EXAMPLE.com"})
	require.Equal(t, []string{"example.com", "www.example.com"}, got)
}

func TestDirectory(t *testing.T) {
	require.Equal(t, DirectoryProd, Directory(false))
	require.Equal(t, DirectoryStaging, Directory(true))
	require.Contains(t, Directory(true), "staging")
}

func TestCertRequestHasSANs(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	domains := []string{"example.com", "www.example.com"}
	der, err := certRequest(key, domains)
	require.NoError(t, err)

	csr, err := x509.ParseCertificateRequest(der)
	require.NoError(t, err)
	require.NoError(t, csr.CheckSignature(), "CSR must be self-consistent")
	require.Equal(t, "example.com", csr.Subject.CommonName)
	require.ElementsMatch(t, domains, csr.DNSNames)
}

func TestKeyToPEMRoundTrips(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	p, err := keyToPEM(key)
	require.NoError(t, err)
	require.Contains(t, p, "BEGIN PRIVATE KEY")

	block, _ := pem.Decode([]byte(p))
	require.NotNil(t, block)
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	require.NoError(t, err)
	ec, ok := parsed.(*ecdsa.PrivateKey)
	require.True(t, ok)
	require.Zero(t, ec.D.Cmp(key.D), "round-tripped key matches")
}

func TestChainPEM(t *testing.T) {
	// Two dummy DER blobs → two CERTIFICATE PEM blocks, in order.
	out, err := chainPEM([][]byte{[]byte("leaf-der"), []byte("intermediate-der")})
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(out, "BEGIN CERTIFICATE"))
	require.True(t, strings.Index(out, pemOf("leaf-der")) < strings.Index(out, pemOf("intermediate-der")), "leaf comes first")
}

func pemOf(s string) string {
	var b strings.Builder
	_ = pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: []byte(s)})
	// return just the base64 body line to search for
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	return lines[1]
}
