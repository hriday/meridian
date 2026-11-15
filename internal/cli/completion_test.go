package cli

import (
	"strings"
	"testing"
)

func TestCompletionScriptsContainCommands(t *testing.T) {
	// Every shell script must mention a real command and the right install hook.
	cases := []struct{ shell, must string }{
		{"bash", "complete -F _meridian_cli"},
		{"zsh", "#compdef meridian-cli"},
		{"fish", "__fish_use_subcommand"},
		{"powershell", "Register-ArgumentCompleter"},
	}
	for _, c := range cases {
		s, err := CompletionScript(c.shell)
		if err != nil {
			t.Fatalf("%s: %v", c.shell, err)
		}
		if !strings.Contains(s, c.must) {
			t.Errorf("%s script missing %q", c.shell, c.must)
		}
		if !strings.Contains(s, "lookup") || !strings.Contains(s, "bgp") {
			t.Errorf("%s script missing known commands", c.shell)
		}
	}
}

func TestCompletionUnknownShell(t *testing.T) {
	if _, err := CompletionScript("tcsh"); err == nil {
		t.Fatal("unknown shell should error")
	}
}

func TestCompletionCoversWholeCatalog(t *testing.T) {
	// The command list must include every catalog command (no drift).
	got := map[string]bool{}
	for _, n := range names() {
		got[n] = true
	}
	for _, g := range Catalog() {
		for _, c := range g.Commands {
			if !got[c.Name] {
				t.Errorf("completion list missing catalog command %q", c.Name)
			}
		}
	}
}

func TestRunCompletionDispatch(t *testing.T) {
	var out, errb strings.Builder
	if code := Run([]string{"completion", "bash"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errb.String())
	}
	if !strings.Contains(out.String(), "complete -F _meridian_cli") {
		t.Error("bash completion not emitted via dispatch")
	}
}
