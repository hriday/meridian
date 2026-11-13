package cli

import (
	"strings"
	"testing"
)

func TestCatalogEntriesComplete(t *testing.T) {
	groups := Catalog()
	if len(groups) < 6 {
		t.Fatalf("expected the full set of groups, got %d", len(groups))
	}
	seen := map[string]bool{}
	for _, g := range groups {
		if g.Name == "" {
			t.Error("a group has no name")
		}
		if len(g.Commands) == 0 {
			t.Errorf("group %q has no commands", g.Name)
		}
		for _, c := range g.Commands {
			if c.Name == "" || c.Desc == "" {
				t.Errorf("group %q has a command with an empty name/desc: %+v", g.Name, c)
			}
			if seen[c.Name] {
				t.Errorf("command %q appears twice in the catalog", c.Name)
			}
			seen[c.Name] = true
		}
	}
}

func TestRenderHelpCoversEveryCatalogCommand(t *testing.T) {
	help := renderHelp()
	for _, g := range Catalog() {
		for _, c := range g.Commands {
			if !strings.Contains(help, c.Name) {
				t.Errorf("rendered help is missing command %q", c.Name)
			}
		}
	}
}
