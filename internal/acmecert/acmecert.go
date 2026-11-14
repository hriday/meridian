// Package acmecert issues Let's Encrypt (ACME) certificates via the DNS-01
// challenge — the firewall-friendly method, since the CA reads a TXT record from
// public DNS and never connects back to your host. It's a one-shot flow: Begin
// returns the TXT record(s) to publish, and once they're live, Finalize
// completes validation and returns the certificate chain and private key.
//
// It deliberately does not store anything or renew: you get a cert now, you
// deploy and renew it. The ACME account key is generated fresh per order, so
// nothing persists on disk. Private keys never touch the logs.
package acmecert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/acme"
)

const (
	// DirectoryProd is Let's Encrypt production (trusted certs, rate-limited).
	DirectoryProd = "https://acme-v02.api.letsencrypt.org/directory"
	// DirectoryStaging issues untrusted test certs with far looser limits.
	DirectoryStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"

	userAgent = "meridian (+https://github.com/hriday/meridian)"
)

// DNSRecord is a TXT record the operator must publish for one domain.
type DNSRecord struct {
	Domain string `json:"domain"`
	Name   string `json:"name"`  // e.g. _acme-challenge.example.com
	Value  string `json:"value"` // the TXT value to set
}

// Order holds the in-flight ACME state between Begin and Finalize. It is not
// serializable; the caller keeps it in memory for the life of the flow.
type Order struct {
	client  *acme.Client
	order   *acme.Order
	pending []pendingAuthz
	domains []string
}

type pendingAuthz struct {
	url  string
	chal *acme.Challenge
}

// challengeName is the TXT record name for a domain's DNS-01 challenge. A
// wildcard validates against its base name (the "*." is dropped).
func challengeName(domain string) string {
	return "_acme-challenge." + strings.TrimPrefix(strings.TrimSpace(domain), "*.")
}

// Begin registers a throwaway ACME account, opens an order for domains, and
// returns the DNS-01 TXT records to publish. dirURL selects prod vs staging.
func Begin(ctx context.Context, dirURL, email string, domains []string) (*Order, []DNSRecord, error) {
	domains = cleanDomains(domains)
	if len(domains) == 0 {
		return nil, nil, fmt.Errorf("enter at least one domain")
	}

	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	client := &acme.Client{Key: accountKey, DirectoryURL: dirURL, UserAgent: userAgent}

	acct := &acme.Account{}
	if email != "" {
		acct.Contact = []string{"mailto:" + email}
	}
	if _, err := client.Register(ctx, acct, acme.AcceptTOS); err != nil {
		return nil, nil, fmt.Errorf("couldn't register with the ACME server: %w", err)
	}

	order, err := client.AuthorizeOrder(ctx, acme.DomainIDs(domains...))
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't create the certificate order: %w", err)
	}

	o := &Order{client: client, order: order, domains: domains}
	var records []DNSRecord
	for _, authzURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return nil, nil, err
		}
		if authz.Status == acme.StatusValid {
			continue // already authorized (nothing to publish)
		}
		chal := dns01Challenge(authz.Challenges)
		if chal == nil {
			return nil, nil, fmt.Errorf("the ACME server offered no dns-01 challenge for %s", authz.Identifier.Value)
		}
		val, err := client.DNS01ChallengeRecord(chal.Token)
		if err != nil {
			return nil, nil, err
		}
		records = append(records, DNSRecord{
			Domain: authz.Identifier.Value,
			Name:   challengeName(authz.Identifier.Value),
			Value:  val,
		})
		o.pending = append(o.pending, pendingAuthz{url: authzURL, chal: chal})
	}
	return o, records, nil
}

// Finalize tells the CA the TXT records are published, waits for validation,
// then issues the certificate. It returns the full chain and the new private
// key, both PEM-encoded.
func (o *Order) Finalize(ctx context.Context) (fullchainPEM, keyPEM string, err error) {
	for _, p := range o.pending {
		if _, err := o.client.Accept(ctx, p.chal); err != nil {
			return "", "", fmt.Errorf("the challenge for %s was rejected: %w", p.chal.URI, err)
		}
	}
	for _, p := range o.pending {
		if _, err := o.client.WaitAuthorization(ctx, p.url); err != nil {
			return "", "", fmt.Errorf("validation failed (check the TXT record is published and has propagated): %w", err)
		}
	}

	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	csr, err := certRequest(certKey, o.domains)
	if err != nil {
		return "", "", err
	}
	der, _, err := o.client.CreateOrderCert(ctx, o.order.FinalizeURL, csr, true)
	if err != nil {
		return "", "", fmt.Errorf("the CA wouldn't issue the certificate: %w", err)
	}

	fullchainPEM, err = chainPEM(der)
	if err != nil {
		return "", "", err
	}
	keyPEM, err = keyToPEM(certKey)
	if err != nil {
		return "", "", err
	}
	return fullchainPEM, keyPEM, nil
}

// Domains returns the order's domains (for the caller's result summary).
func (o *Order) Domains() []string { return o.domains }

func dns01Challenge(challenges []*acme.Challenge) *acme.Challenge {
	for _, c := range challenges {
		if c.Type == "dns-01" {
			return c
		}
	}
	return nil
}

// certRequest builds a DER-encoded CSR for key covering all domains (SAN).
func certRequest(key *ecdsa.PrivateKey, domains []string) ([]byte, error) {
	tmpl := &x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: domains[0]},
		DNSNames: domains,
	}
	return x509.CreateCertificateRequest(rand.Reader, tmpl, key)
}

// chainPEM encodes a DER certificate chain as concatenated PEM blocks.
func chainPEM(der [][]byte) (string, error) {
	var b strings.Builder
	for _, c := range der {
		if err := pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: c}); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

// keyToPEM encodes an EC private key as a PKCS#8 PEM block.
func keyToPEM(key *ecdsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := pem.Encode(&b, &pem.Block{Type: "PRIVATE KEY", Bytes: der}); err != nil {
		return "", err
	}
	return b.String(), nil
}

// cleanDomains trims, lowercases, and drops blank/duplicate domains.
func cleanDomains(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		d = strings.TrimSuffix(d, ".")
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

// Directory returns the ACME directory URL for prod or staging.
func Directory(staging bool) string {
	if staging {
		return DirectoryStaging
	}
	return DirectoryProd
}
