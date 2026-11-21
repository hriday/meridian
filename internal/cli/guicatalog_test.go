package cli

import (
	"bytes"
	"strings"
	"testing"
)

// knownGroups is the set of sidebar group names the GUI uses; the overview must
// stay within it so a typo'd group can't silently create an orphan section.
var knownGroups = map[string]bool{
	"Internet":            true,
	"Addressing":          true,
	"Local / environment": true,
	"Local network":       true,
	"SNMP & monitoring":   true,
	"Collectors":          true,
	"Security":            true,
	"Config insight":      true,
}

func TestGUICatalogIsComplete(t *testing.T) {
	groups := GUICatalog()
	if len(groups) == 0 {
		t.Fatal("GUICatalog is empty")
	}
	seenGroup := map[string]bool{}
	seenTool := map[string]bool{}
	total := 0
	for _, g := range groups {
		if !knownGroups[g.Name] {
			t.Errorf("group %q is not a known sidebar group", g.Name)
		}
		if seenGroup[g.Name] {
			t.Errorf("group %q appears twice", g.Name)
		}
		seenGroup[g.Name] = true
		if len(g.Tools) == 0 {
			t.Errorf("group %q has no tools", g.Name)
		}
		for _, tool := range g.Tools {
			if strings.TrimSpace(tool.Name) == "" {
				t.Errorf("group %q has a tool with no name", g.Name)
			}
			if strings.TrimSpace(tool.Desc) == "" {
				t.Errorf("tool %q in %q has no description", tool.Name, g.Name)
			}
			if seenTool[tool.Name] {
				t.Errorf("tool %q appears twice", tool.Name)
			}
			seenTool[tool.Name] = true
			total++
		}
	}
	// The app ships ~90 tools; guard against a group being dropped wholesale.
	if total < 80 {
		t.Errorf("GUICatalog lists only %d tools, far fewer than the app ships", total)
	}
}

// TestGUIReferenceCoversNewTools pins that the tools added this cycle — the ones
// most likely to be forgotten in the hand-kept overview — are actually listed,
// so the overview can't quietly fall behind the sidebar.
func TestGUIReferenceCoversNewTools(t *testing.T) {
	want := []string{"Link inspector", "Connected peripherals", "Tools overview", "Meshtastic mesh", "Bluetooth LE survey"}
	have := map[string]bool{}
	for _, g := range GUICatalog() {
		for _, tool := range g.Tools {
			have[tool.Name] = true
		}
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("GUICatalog is missing tool %q", w)
		}
	}
}

func TestGUIReferencePDFRenders(t *testing.T) {
	b, err := RenderGUIReferencePDF()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Errorf("output is not a PDF (first bytes: %q)", b[:min(8, len(b))])
	}
	if len(b) < 2000 {
		t.Errorf("PDF is implausibly small (%d bytes) for the tools overview", len(b))
	}
}
