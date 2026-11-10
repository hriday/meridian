package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/whois"
)

func TestPrintWhoisResult(t *testing.T) {
	var buf bytes.Buffer
	printWhoisResult(&buf, whois.Result{
		Object: "example.com", Server: "whois.verisign-grs.com",
		Chain: []string{"whois.iana.org", "whois.verisign-grs.com"},
		Raw:   "Domain Name: EXAMPLE.COM\nRegistrar: Example Registrar",
	})
	out := buf.String()
	for _, want := range []string{"whois.iana.org", "whois.verisign-grs.com", "Example Registrar"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRunWhoisMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runWhois(nil, &out, &errb); code != 2 {
		t.Errorf("runWhois with no args: got exit %d, want 2", code)
	}
}
