package cli

import (
	"strings"
	"testing"
)

func TestRunbooksList(t *testing.T) {
	if len(Runbooks()) < 2 {
		t.Fatalf("want at least 2 profiles, got %d", len(Runbooks()))
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
