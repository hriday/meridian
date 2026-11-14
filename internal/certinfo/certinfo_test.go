package certinfo

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func selfSignedPEM(t *testing.T) string {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(12345),
		Subject:      pkix.Name{CommonName: "test.example.com", Organization: []string{"Acme"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(48 * time.Hour),
		DNSNames:     []string{"test.example.com", "www.example.com"},
		IPAddresses:  []net.IP{net.ParseIP("192.0.2.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func csrPEM(t *testing.T) string {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: "csr.example.com"},
		DNSNames: []string{"csr.example.com", "alt.example.com"},
	}
	der, err := x509.CreateCertificateRequest(rand.Reader, tmpl, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestDecodeCertificate(t *testing.T) {
	res, err := Decode(selfSignedPEM(t), time.Now())
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(res.Items))
	}
	it := res.Items[0]
	if it.Kind != "certificate" {
		t.Errorf("kind = %q", it.Kind)
	}
	if !strings.Contains(it.Subject, "test.example.com") {
		t.Errorf("subject = %q", it.Subject)
	}
	if !it.SelfSigned {
		t.Error("should be self-signed (subject == issuer)")
	}
	if it.Expired {
		t.Error("cert valid 48h should not be expired")
	}
	if len(it.DNSNames) != 2 || it.IPs[0] != "192.0.2.1" {
		t.Errorf("SANs wrong: dns=%v ip=%v", it.DNSNames, it.IPs)
	}
	if !strings.HasPrefix(it.KeyType, "EC ") {
		t.Errorf("keyType = %q, want EC", it.KeyType)
	}
	if len(it.KeyUsage) == 0 || len(it.ExtKeyUsage) == 0 {
		t.Errorf("usages empty: %v / %v", it.KeyUsage, it.ExtKeyUsage)
	}
}

func TestDecodeExpiredCert(t *testing.T) {
	res, err := Decode(selfSignedPEM(t), time.Now().Add(72*time.Hour)) // past NotAfter
	if err != nil {
		t.Fatal(err)
	}
	if !res.Items[0].Expired {
		t.Error("evaluating 'now' after NotAfter should read expired")
	}
}

func TestDecodeCSR(t *testing.T) {
	res, err := Decode(csrPEM(t), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	it := res.Items[0]
	if it.Kind != "csr" {
		t.Errorf("kind = %q, want csr", it.Kind)
	}
	if !strings.Contains(it.Subject, "csr.example.com") || len(it.DNSNames) != 2 {
		t.Errorf("CSR parsed wrong: subj=%q dns=%v", it.Subject, it.DNSNames)
	}
}

func TestDecodeChainAndSkipsKeys(t *testing.T) {
	// A cert plus a private-key block: the key must be ignored, the cert parsed.
	combined := selfSignedPEM(t) + "-----BEGIN PRIVATE KEY-----\nMIIBogIBADANBg\n-----END PRIVATE KEY-----\n" + selfSignedPEM(t)
	res, err := Decode(combined, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 {
		t.Errorf("want 2 certs (key skipped), got %d", len(res.Items))
	}
}

func TestDecodeNothing(t *testing.T) {
	if _, err := Decode("not a pem", time.Now()); err == nil {
		t.Error("garbage input should error")
	}
}
