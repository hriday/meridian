package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/certexpiry"
	"github.com/hriday/meridian/internal/ctlog"
	"github.com/hriday/meridian/internal/cve"
	"github.com/hriday/meridian/internal/hashid"
	"github.com/hriday/meridian/internal/secheaders"
	"github.com/hriday/meridian/internal/threatintel"
)

func TestPrintCVE(t *testing.T) {
	var buf bytes.Buffer
	printCVE(&buf, cve.CVE{
		ID: "CVE-2021-44228", Published: "2021-12-10", Severity: "CRITICAL", CVSS: 10.0,
		Vector: "CVSS:3.1/AV:N", CWE: []string{"CWE-917"}, KEV: true,
		Description: "Apache Log4j2 JNDI RCE.", References: []string{"https://logging.apache.org/"},
	})
	out := buf.String()
	for _, want := range []string{"CVE-2021-44228", "CRITICAL", "10.0", "CWE-917", "KEV", "Log4j2", "logging.apache.org"} {
		if !strings.Contains(out, want) {
			t.Errorf("printCVE output missing %q\n---\n%s", want, out)
		}
	}
}

func TestPrintCVEList(t *testing.T) {
	var buf bytes.Buffer
	printCVEList(&buf, []cve.CVE{
		{ID: "CVE-2021-44228", Severity: "CRITICAL", KEV: true, Description: "Log4Shell"},
		{ID: "CVE-2014-0160", Severity: "HIGH", Description: "Heartbleed"},
	})
	out := buf.String()
	for _, want := range []string{"CVE-2021-44228", "KEV", "Log4Shell", "CVE-2014-0160", "Heartbleed"} {
		if !strings.Contains(out, want) {
			t.Errorf("printCVEList missing %q\n%s", want, out)
		}
	}
}

func TestPrintHashID(t *testing.T) {
	var buf bytes.Buffer
	printHashID(&buf, []hashid.Candidate{
		{Name: "NTLM", Mode: 1000, Note: "Windows/AD hash"},
		{Name: "Argon2", Mode: -1},
	})
	out := buf.String()
	for _, want := range []string{"NTLM", "hashcat -m 1000", "Windows/AD", "Argon2", "no hashcat mode"} {
		if !strings.Contains(out, want) {
			t.Errorf("printHashID missing %q\n%s", want, out)
		}
	}
}

func TestPrintThreatIntel(t *testing.T) {
	var buf bytes.Buffer
	printThreatIntel(&buf, []threatintel.Hit{
		{IP: "91.240.118.168", Listed: true, Malware: "Dridex", Status: "online", Country: "RU", LastOnline: "2026-11-10"},
		{IP: "8.8.8.8", Listed: false},
	})
	out := buf.String()
	for _, want := range []string{"91.240.118.168", "BOTNET C2", "Dridex", "RU", "8.8.8.8", "clean"} {
		if !strings.Contains(out, want) {
			t.Errorf("printThreatIntel missing %q\n%s", want, out)
		}
	}
}

func TestPrintCertExpiry(t *testing.T) {
	var buf bytes.Buffer
	printCertExpiry(&buf, []certexpiry.Result{
		{Host: "expired.example:443", OK: true, Expired: true, DaysLeft: -12, NotAfter: "2025-12-20", Issuer: "Old CA"},
		{Host: "soon.example:443", OK: true, DaysLeft: 10, NotAfter: "2026-01-11", Issuer: "Let's Encrypt"},
		{Host: "dead.example:443", OK: false, Error: "connection refused"},
	})
	out := buf.String()
	for _, want := range []string{"expired.example", "EXPIRED 12d ago", "soon.example", "10 days left", "Let's Encrypt", "dead.example", "unreachable", "connection refused"} {
		if !strings.Contains(out, want) {
			t.Errorf("printCertExpiry missing %q\n%s", want, out)
		}
	}
}

func TestPrintSecHeaders(t *testing.T) {
	var buf bytes.Buffer
	printSecHeaders(&buf, secheaders.Report{
		FinalURL: "https://example.com/", Score: 72, Grade: "B",
		Headers: []secheaders.Header{
			{Name: "Strict-Transport-Security", Present: true, Grade: "a", OWASP: "A02 Cryptographic Failures"},
			{Name: "Content-Security-Policy", Present: false, Grade: "f", OWASP: "A03 Injection", Note: "no CSP set"},
		},
	})
	out := buf.String()
	for _, want := range []string{"example.com", "72/100", "grade B", "Strict-Transport-Security", "A02", "Content-Security-Policy", "no CSP set"} {
		if !strings.Contains(out, want) {
			t.Errorf("printSecHeaders missing %q\n%s", want, out)
		}
	}
}

func TestPrintSubdomains(t *testing.T) {
	var buf bytes.Buffer
	printSubdomains(&buf, []ctlog.Subdomain{
		{Name: "api.example.com", Certs: 4, FirstSeen: "2023-01-01", LastSeen: "2026-10-01"},
		{Name: "example.com", Wildcard: true, Certs: 9, FirstSeen: "2020-01-01", LastSeen: "2026-11-01"},
	})
	out := buf.String()
	for _, want := range []string{"2 subdomains", "api.example.com", "*.example.com", "4 certs"} {
		if !strings.Contains(out, want) {
			t.Errorf("printSubdomains missing %q\n%s", want, out)
		}
	}
}

// Each command exits 2 (usage error) when given no positional argument.
func TestSecurityCommandsMissingArg(t *testing.T) {
	cases := map[string]func([]string) int{
		"cve":         func(a []string) int { var o, e bytes.Buffer; return runCVE(a, &o, &e) },
		"hashid":      func(a []string) int { var o, e bytes.Buffer; return runHashID(a, &o, &e) },
		"threatintel": func(a []string) int { var o, e bytes.Buffer; return runThreatIntel(a, &o, &e) },
		"certexpiry":  func(a []string) int { var o, e bytes.Buffer; return runCertExpiry(a, &o, &e) },
		"sechdr":      func(a []string) int { var o, e bytes.Buffer; return runSecHeaders(a, &o, &e) },
		"subdomains":  func(a []string) int { var o, e bytes.Buffer; return runSubdomains(a, &o, &e) },
	}
	for name, fn := range cases {
		if code := fn(nil); code != 2 {
			t.Errorf("%s with no args: got exit %d, want 2", name, code)
		}
	}
}
