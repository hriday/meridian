package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunbooksList(t *testing.T) {
	if len(Runbooks()) < 2 {
		t.Fatalf("want at least 2 profiles, got %d", len(Runbooks()))
	}
}

func TestRenderRunbookFormats(t *testing.T) {
	d := RunbookData{
		Profile: "org-recon", Name: "Org Recon", Target: "example.com",
		Steps: []RunbookStep{{Command: "whois", Output: "registrar: Example", OK: true}},
	}
	if md := RenderRunbook(d, "md"); !strings.Contains(md, "# Runbook") || !strings.Contains(md, "whois") {
		t.Errorf("md render: %q", md)
	}
	if js := RenderRunbook(d, "json"); !strings.Contains(js, "\"profile\": \"org-recon\"") {
		t.Errorf("json render: %q", js)
	}
	if h := RenderRunbook(d, "html"); !strings.Contains(h, "<h1>") || !strings.Contains(h, "whois") {
		t.Errorf("html render: %q", h)
	}
}

func TestBuiltinProfileCommandsAreReal(t *testing.T) {
	// Every built-in recipe's steps must reference real subcommands (guards
	// against a typo'd cmd in the recipe book that would only fail at run time).
	known := map[string]bool{}
	for _, n := range names() { // from the catalog (completion.go)
		known[n] = true
	}
	for _, p := range rbProfiles {
		if len(p.steps) == 0 {
			t.Errorf("profile %q has no steps", p.id)
		}
		for _, s := range p.steps {
			if !known[s.cmd] {
				t.Errorf("profile %q step %q is not a known command", p.id, s.cmd)
			}
		}
	}
	// org-recon should be the comprehensive one.
	if p, ok := findProfile("org-recon"); !ok || len(p.steps) < 8 {
		t.Error("org-recon should exist with a comprehensive step list")
	}
}

func TestRunbookUnknownProfile(t *testing.T) {
	msg, code := RunbookReport("nope", "x", "md")
	if code != 2 {
		t.Fatalf("want code 2 for unknown profile, got %d", code)
	}
	if !strings.Contains(msg, "unknown runbook") {
		t.Fatalf("expected an explanatory message, got %q", msg)
	}
}

func TestRenderRunbookMD(t *testing.T) {
	d := RunbookData{Name: "Audit Domain", Target: "acme.com", Generated: "T", Steps: []RunbookStep{
		{Command: "lookup", Output: "hello", OK: true},
		{Command: "cipherscan", Output: "bad", OK: false},
	}}
	md := renderRunbookMD(d)
	if !strings.Contains(md, "# Runbook: Audit Domain — acme.com") {
		t.Fatal("missing title")
	}
	if !strings.Contains(md, "## lookup") || !strings.Contains(md, "```\nhello\n```") {
		t.Fatal("missing step section/output")
	}
	if !strings.Contains(md, "## cipherscan ⚠️") {
		t.Fatal("failed step not marked")
	}
}

func TestRenderRunbookHTMLEscapes(t *testing.T) {
	d := RunbookData{Name: "X", Target: "t", Steps: []RunbookStep{
		{Command: "c", Output: "<script>alert(1)</script>", OK: true},
	}}
	h := renderRunbookHTML(d)
	if strings.Contains(h, "<script>alert(1)</script>") {
		t.Fatal("tool output must be HTML-escaped in the report")
	}
	if !strings.Contains(h, "&lt;script&gt;") {
		t.Fatal("expected escaped output")
	}
}

func TestLoadProfileYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myaudit.yaml")
	content := "name: My Audit\ndesc: custom\nsteps:\n  - cmd: lookup\n    args: [\"{t}\"]\n  - cmd: dnsaudit\n    args: [\"{t}\"]\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := loadProfileFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.name != "My Audit" || len(p.steps) != 2 {
		t.Fatalf("got name=%q steps=%d", p.name, len(p.steps))
	}
	if p.steps[0].cmd != "lookup" || p.steps[1].cmd != "dnsaudit" {
		t.Fatalf("steps = %+v", p.steps)
	}
	if len(p.steps[0].args) != 1 || p.steps[0].args[0] != "{t}" {
		t.Fatalf("args = %v", p.steps[0].args)
	}
}

func TestLoadProfileTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hostcheck.toml")
	content := "name = \"Host Check\"\n\n[[steps]]\ncmd = \"ping\"\nargs = [\"{t}\", \"--count\", \"3\"]\n\n[[steps]]\ncmd = \"tls\"\nargs = [\"{t}\"]\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := loadProfileFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.name != "Host Check" || len(p.steps) != 2 {
		t.Fatalf("got name=%q steps=%d", p.name, len(p.steps))
	}
	if p.steps[0].cmd != "ping" || len(p.steps[0].args) != 3 {
		t.Fatalf("step0 = %+v", p.steps[0])
	}
}

func TestLoadProfileRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("name: nope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadProfileFile(path); err == nil {
		t.Fatal("profile with no steps should error")
	}
}

func TestRunbookFileInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my.yaml")
	if err := os.WriteFile(path, []byte("name: My Recipe\nsteps:\n  - cmd: lookup\n    args: [\"{t}\"]\n  - cmd: whois\n    args: [\"{t}\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := RunbookFileInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "My Recipe" || info.Steps != 2 {
		t.Fatalf("info = %+v", info)
	}
	if _, err := RunbookFileInfo(filepath.Join(dir, "nope.yaml")); err == nil {
		t.Fatal("missing file should error")
	}
}

func TestResolveProfileBuiltinVsFile(t *testing.T) {
	// A built-in id resolves without touching disk.
	if _, ok, err := resolveProfile("audit-domain"); err != nil || !ok {
		t.Fatalf("builtin: ok=%v err=%v", ok, err)
	}
	// A path that looks like a file but doesn't exist is an error, not "unknown".
	if _, _, err := resolveProfile("does-not-exist.yaml"); err == nil {
		t.Fatal("missing file should error")
	}
}
