package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/certinfo"
)

func TestPrintCertInfo(t *testing.T) {
	var buf bytes.Buffer
	na := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
	printCertInfo(&buf, certinfo.Result{Items: []certinfo.Item{{
		Kind: "certificate", Subject: "CN=test.example.com", Issuer: "CN=test.example.com",
		SelfSigned: true, KeyType: "EC P-256", SigAlg: "ECDSA-SHA256", NotAfter: &na, DaysLeft: 100,
		DNSNames: []string{"test.example.com", "www.example.com"}, KeyUsage: []string{"Digital Signature"},
	}}})
	out := buf.String()
	for _, want := range []string{"Certificate", "test.example.com", "self-signed", "EC P-256", "2030-06-01", "www.example.com", "Digital Signature"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintCertInfoCSR(t *testing.T) {
	var buf bytes.Buffer
	printCertInfo(&buf, certinfo.Result{Items: []certinfo.Item{{
		Kind: "csr", Subject: "CN=csr.example.com", KeyType: "RSA 2048-bit", SigAlg: "SHA256-RSA",
	}}})
	if !strings.Contains(buf.String(), "Signing Request") {
		t.Errorf("CSR should be labelled: %s", buf.String())
	}
}
