// Package certinfo decodes pasted PEM certificates and CSRs into a readable
// summary — locally, so an internal cert or a CSR never has to be pasted into a
// web "cert decoder". It complements the tls tool (which fetches a live server's
// cert) by working on PEM text you already have. Point-in-time; nothing stored.
package certinfo

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Item is one decoded certificate or CSR.
type Item struct {
	Kind         string     `json:"kind"` // "certificate" or "csr"
	Subject      string     `json:"subject"`
	Issuer       string     `json:"issuer,omitempty"`
	SerialHex    string     `json:"serialHex,omitempty"`
	NotBefore    *time.Time `json:"notBefore,omitempty"`
	NotAfter     *time.Time `json:"notAfter,omitempty"`
	DaysLeft     int        `json:"daysLeft,omitempty"`
	Expired      bool       `json:"expired,omitempty"`
	SelfSigned   bool       `json:"selfSigned,omitempty"`
	IsCA         bool       `json:"isCA,omitempty"`
	DNSNames     []string   `json:"dnsNames,omitempty"`
	IPs          []string   `json:"ips,omitempty"`
	Emails       []string   `json:"emails,omitempty"`
	URIs         []string   `json:"uris,omitempty"`
	KeyType      string     `json:"keyType"`
	SigAlg       string     `json:"sigAlg"`
	KeyUsage     []string   `json:"keyUsage,omitempty"`
	ExtKeyUsage  []string   `json:"extKeyUsage,omitempty"`
	Version      int        `json:"version,omitempty"`
}

// Result is every certificate/CSR found in the input, in order.
type Result struct {
	Items []Item `json:"items"`
}

// Decode parses every CERTIFICATE / CERTIFICATE REQUEST PEM block in text.
// Private-key blocks are skipped (never decoded).
func Decode(text string, now time.Time) (Result, error) {
	rest := []byte(text)
	var res Result
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			if c, err := x509.ParseCertificate(block.Bytes); err == nil {
				res.Items = append(res.Items, fromCert(c, now))
			}
		case "CERTIFICATE REQUEST", "NEW CERTIFICATE REQUEST":
			if r, err := x509.ParseCertificateRequest(block.Bytes); err == nil {
				res.Items = append(res.Items, fromCSR(r))
			}
		}
	}
	if len(res.Items) == 0 {
		return Result{}, errors.New("no certificate or CSR found — paste PEM text beginning with -----BEGIN CERTIFICATE----- or -----BEGIN CERTIFICATE REQUEST-----")
	}
	return res, nil
}

func fromCert(c *x509.Certificate, now time.Time) Item {
	nb, na := c.NotBefore, c.NotAfter
	it := Item{
		Kind:        "certificate",
		Subject:     c.Subject.String(),
		Issuer:      c.Issuer.String(),
		SerialHex:   strings.ToUpper(c.SerialNumber.Text(16)),
		NotBefore:   &nb,
		NotAfter:    &na,
		DaysLeft:    int(na.Sub(now).Hours() / 24),
		Expired:     now.After(na),
		SelfSigned:  c.Subject.String() == c.Issuer.String(),
		IsCA:        c.IsCA,
		DNSNames:    c.DNSNames,
		Emails:      c.EmailAddresses,
		KeyType:     keyType(c.PublicKey),
		SigAlg:      c.SignatureAlgorithm.String(),
		KeyUsage:    keyUsages(c.KeyUsage),
		ExtKeyUsage: extKeyUsages(c.ExtKeyUsage),
		Version:     c.Version,
	}
	for _, ip := range c.IPAddresses {
		it.IPs = append(it.IPs, ip.String())
	}
	for _, u := range c.URIs {
		it.URIs = append(it.URIs, u.String())
	}
	return it
}

func fromCSR(r *x509.CertificateRequest) Item {
	it := Item{
		Kind:     "csr",
		Subject:  r.Subject.String(),
		DNSNames: r.DNSNames,
		Emails:   r.EmailAddresses,
		KeyType:  keyType(r.PublicKey),
		SigAlg:   r.SignatureAlgorithm.String(),
	}
	for _, ip := range r.IPAddresses {
		it.IPs = append(it.IPs, ip.String())
	}
	for _, u := range r.URIs {
		it.URIs = append(it.URIs, u.String())
	}
	return it
}

func keyType(pub any) string {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d-bit", k.N.BitLen())
	case *ecdsa.PublicKey:
		return "EC " + k.Curve.Params().Name
	case ed25519.PublicKey:
		return "Ed25519"
	}
	return "unknown"
}

func keyUsages(u x509.KeyUsage) []string {
	names := []struct {
		bit  x509.KeyUsage
		name string
	}{
		{x509.KeyUsageDigitalSignature, "Digital Signature"},
		{x509.KeyUsageContentCommitment, "Content Commitment"},
		{x509.KeyUsageKeyEncipherment, "Key Encipherment"},
		{x509.KeyUsageDataEncipherment, "Data Encipherment"},
		{x509.KeyUsageKeyAgreement, "Key Agreement"},
		{x509.KeyUsageCertSign, "Certificate Sign"},
		{x509.KeyUsageCRLSign, "CRL Sign"},
	}
	var out []string
	for _, n := range names {
		if u&n.bit != 0 {
			out = append(out, n.name)
		}
	}
	return out
}

func extKeyUsages(us []x509.ExtKeyUsage) []string {
	name := map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageServerAuth:      "Server Auth",
		x509.ExtKeyUsageClientAuth:      "Client Auth",
		x509.ExtKeyUsageCodeSigning:     "Code Signing",
		x509.ExtKeyUsageEmailProtection: "Email Protection",
		x509.ExtKeyUsageTimeStamping:    "Time Stamping",
		x509.ExtKeyUsageOCSPSigning:     "OCSP Signing",
	}
	var out []string
	for _, u := range us {
		if n, ok := name[u]; ok {
			out = append(out, n)
		}
	}
	return out
}
