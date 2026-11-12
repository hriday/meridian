package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/ciscopw"
	"github.com/hriday/meridian/internal/configdiff"
	"github.com/hriday/meridian/internal/hibp"
	"github.com/hriday/meridian/internal/tfscan"
)

func TestPrintCiscoSecrets(t *testing.T) {
	var buf bytes.Buffer
	printCiscoSecrets(&buf, []ciscopw.Secret{
		{Context: "username bob", Keyword: "password", Type: "7", Encoded: "0822455D0A16", Plaintext: "cisco", Decoded: true, Verdict: "reversible"},
		{Context: "enable secret", Keyword: "secret", Type: "5", Encoded: "$1$abc$xyz", Verdict: "weak-hash"},
	})
	out := buf.String()
	for _, want := range []string{"reversible", "username bob", "recovered: cisco", "weak-hash", "enable secret", "$1$abc$xyz"} {
		if !strings.Contains(out, want) {
			t.Errorf("printCiscoSecrets missing %q\n%s", want, out)
		}
	}
}

func TestPrintCiscoSecretsEmpty(t *testing.T) {
	var buf bytes.Buffer
	printCiscoSecrets(&buf, nil)
	if !strings.Contains(buf.String(), "no password") {
		t.Errorf("empty case should say none found: %q", buf.String())
	}
}

func TestRunCiscoPWDecode(t *testing.T) {
	// --decode runs offline; a known type-7 vector decodes to "cisco".
	var out, errb bytes.Buffer
	if code := runCiscoPW([]string{"--decode", "0822455D0A16"}, &out, &errb); code != 0 {
		t.Fatalf("decode exit %d, stderr=%s", code, errb.String())
	}
	if got := strings.TrimSpace(out.String()); got != "cisco" {
		t.Errorf("decoded %q, want cisco", got)
	}
}

func TestPrintTFScan(t *testing.T) {
	var buf bytes.Buffer
	printTFScan(&buf, tfscan.Analysis{
		Source: "hcl",
		Counts: tfscan.Counts{Resources: 5, SecurityGroups: 2, Exposed: 1, Critical: 1, Findings: 3},
		Findings: []tfscan.Finding{
			{Severity: "critical", Title: "SSH open to 0.0.0.0/0", Detail: "port 22 world-open", Resource: "aws_security_group.web", File: "main.tf", Line: 12},
		},
		Notes: []string{"dynamic value unresolved"},
	})
	out := buf.String()
	for _, want := range []string{"source hcl", "5 resources", "1 critical", "CRITICAL", "SSH open", "aws_security_group.web", "main.tf:12", "dynamic value"} {
		if !strings.Contains(out, want) {
			t.Errorf("printTFScan missing %q\n%s", want, out)
		}
	}
}

func TestPrintConfigDiff(t *testing.T) {
	var buf bytes.Buffer
	printConfigDiff(&buf, configdiff.Result{
		Added: 1, Removed: 1, Sections: []string{"interface Gi0/1"},
		Lines: []configdiff.Line{
			{Op: configdiff.OpDel, Text: " description OLD", Section: "interface Gi0/1"},
			{Op: configdiff.OpAdd, Text: " description NEW", Section: "interface Gi0/1"},
			{Op: configdiff.OpSame, Text: "hostname R1"},
		},
	})
	out := buf.String()
	for _, want := range []string{"1 added, 1 removed", "interface Gi0/1", "- ", "description OLD", "+ ", "description NEW"} {
		if !strings.Contains(out, want) {
			t.Errorf("printConfigDiff missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "hostname R1") {
		t.Errorf("unchanged lines should not be printed:\n%s", out)
	}
}

func TestPrintBreachNames(t *testing.T) {
	var buf bytes.Buffer
	printBreachNames(&buf, "a@b.com", []string{"Adobe", "LinkedIn"})
	out := buf.String()
	for _, want := range []string{"a@b.com", "2 breach", "Adobe", "LinkedIn", "no k-anonymity"} {
		if !strings.Contains(out, want) {
			t.Errorf("printBreachNames missing %q\n%s", want, out)
		}
	}

	var clean bytes.Buffer
	printBreachNames(&clean, "safe@b.com", nil)
	if !strings.Contains(clean.String(), "no breaches found") {
		t.Errorf("clean case: %q", clean.String())
	}
}

func TestPrintBreach(t *testing.T) {
	var buf bytes.Buffer
	printBreach(&buf, hibp.Breach{
		Title: "Adobe", Domain: "adobe.com", BreachDate: "2013-10-04", PwnCount: 152445165,
		DataClasses: []string{"Email addresses", "Passwords"}, Description: "In October 2013, Adobe...",
	})
	out := buf.String()
	for _, want := range []string{"Adobe", "adobe.com", "2013-10-04", "152445165", "Passwords", "October 2013"} {
		if !strings.Contains(out, want) {
			t.Errorf("printBreach missing %q\n%s", want, out)
		}
	}
}

func TestConfigInsightMissingArg(t *testing.T) {
	cases := map[string]func([]string) int{
		"cfgdiff":  func(a []string) int { var o, e bytes.Buffer; return runCfgDiff(a, &o, &e) },
		"breaches": func(a []string) int { var o, e bytes.Buffer; return runBreaches(a, &o, &e) },
	}
	for name, fn := range cases {
		if code := fn(nil); code != 2 {
			t.Errorf("%s with no args: got exit %d, want 2", name, code)
		}
	}
	// cfgdiff with one file is still a usage error.
	var o, e bytes.Buffer
	if code := runCfgDiff([]string{"only-one"}, &o, &e); code != 2 {
		t.Errorf("cfgdiff with one arg: got %d, want 2", code)
	}
}
