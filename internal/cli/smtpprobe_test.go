package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/smtpprobe"
)

func TestPrintSMTP(t *testing.T) {
	var buf bytes.Buffer
	printSMTP(&buf, smtpprobe.Result{
		Host: "mx.example.com", Port: "25", Banner: "220 mx.example.com ESMTP",
		Greeting: "mx.example.com", StartTLS: true,
		Capabilities: []string{"SIZE 10240000", "STARTTLS", "8BITMIME"},
		AuthMechs:    []string{"PLAIN", "LOGIN"},
		TLS: &smtpprobe.TLSInfo{
			Version: "TLS 1.3", Cipher: "TLS_AES_128_GCM_SHA256",
			Subject: "mx.example.com", Issuer: "Let's Encrypt",
			DaysLeft: 42, HostMatch: true, NotAfter: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	})
	out := buf.String()
	for _, want := range []string{"mx.example.com", "STARTTLS offered", "TLS 1.3", "Let's Encrypt", "42 days", "valid for this host", "PLAIN"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunSMTPMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runSMTP(nil, &o, &e); code != 2 {
		t.Errorf("smtp with no args: got exit %d, want 2", code)
	}
}
