package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDocsMarkdownCoversEveryCommandAndRecipe(t *testing.T) {
	md := renderDocsMarkdown()
	for _, g := range Catalog() {
		if !strings.Contains(md, "## "+g.Name) {
			t.Errorf("markdown reference is missing group %q", g.Name)
		}
		for _, c := range g.Commands {
			if !strings.Contains(md, "`"+c.Name+"`") {
				t.Errorf("markdown reference is missing command %q", c.Name)
			}
		}
	}
	for _, p := range rbProfiles {
		if !strings.Contains(md, "`"+p.id+"`") {
			t.Errorf("markdown reference is missing recipe %q", p.id)
		}
	}
}

func TestDocsHTMLIsWellFormedEnough(t *testing.T) {
	h := renderDocsHTML()
	if !strings.HasPrefix(h, "<!doctype html>") || !strings.Contains(h, "</html>") {
		t.Error("HTML reference should be a complete document")
	}
	if !strings.Contains(h, "<table>") || !strings.Contains(h, "linkpath") {
		t.Error("HTML reference should contain command tables")
	}
	// A description with special chars must be escaped, not raw.
	if strings.Contains(h, "<command>") {
		t.Error("HTML reference leaked an unescaped angle bracket from a usage string")
	}
}

func TestDocsPDFRenders(t *testing.T) {
	b, err := RenderDocsPDF()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Errorf("output is not a PDF (first bytes: %q)", b[:min(8, len(b))])
	}
	if len(b) < 2000 {
		t.Errorf("PDF is implausibly small (%d bytes) for a full reference", len(b))
	}
}

func TestMdCellEscapesPipes(t *testing.T) {
	if got := mdCell("a|b"); got != "a\\|b" {
		t.Errorf("mdCell(%q) = %q, want pipes escaped", "a|b", got)
	}
}
