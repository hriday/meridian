package cli

import (
	"bytes"
	"os"
	"regexp"
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

// sidebarToolRow matches one entry in Workbench.svelte's tools[] array, capturing
// the friendly name and its group — the two things the overview must mirror.
var sidebarToolRow = regexp.MustCompile(`\{\s*id:\s*"[^"]+",\s*name:\s*"([^"]+)",\s*group:\s*"([^"]+)"`)

// TestGUICatalogMatchesSidebar enforces the feature's central promise: the
// overview page and PDF (GUICatalog, in Go) must list exactly the tools the
// workbench sidebar does (Workbench.svelte, in Svelte), same names and groups.
// Without this, the two hand-kept lists drift the moment someone adds or renames
// a tool in one but not the other — the "never goes stale" claim made real.
func TestGUICatalogMatchesSidebar(t *testing.T) {
	src, err := os.ReadFile("../../frontend/src/lib/Workbench.svelte")
	if err != nil {
		t.Fatalf("reading the sidebar source: %v", err)
	}
	sidebar := map[string]string{} // tool name -> group
	for _, m := range sidebarToolRow.FindAllStringSubmatch(string(src), -1) {
		sidebar[m[1]] = m[2]
	}
	if len(sidebar) < 80 {
		t.Fatalf("parsed only %d tools from the sidebar — the regex has likely drifted from the source", len(sidebar))
	}

	catalog := map[string]string{} // tool name -> group
	for _, g := range GUICatalog() {
		for _, tool := range g.Tools {
			catalog[tool.Name] = g.Name
		}
	}

	for name, group := range sidebar {
		cg, ok := catalog[name]
		if !ok {
			t.Errorf("sidebar tool %q (group %q) is missing from the Tools overview", name, group)
			continue
		}
		if cg != group {
			t.Errorf("tool %q is in group %q in the sidebar but %q in the overview", name, group, cg)
		}
	}
	for name := range catalog {
		if _, ok := sidebar[name]; !ok {
			t.Errorf("Tools overview lists %q, which is not a tool in the sidebar", name)
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
