package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// commandList returns every subcommand name with its one-line description,
// sorted, from the single-source-of-truth Catalog — so completions never drift
// from the real command set.
func commandList() []CommandDoc {
	seen := map[string]bool{}
	var out []CommandDoc
	add := func(c CommandDoc) {
		if !seen[c.Name] {
			seen[c.Name] = true
			out = append(out, c)
		}
	}
	for _, g := range Catalog() {
		for _, c := range g.Commands {
			add(c)
		}
	}
	add(CommandDoc{Name: "help", Desc: "show help"}) // dispatch-only, not in catalog
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func names() []string {
	cmds := commandList()
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.Name
	}
	return out
}

// CompletionScript returns a shell completion script for bash, zsh, fish or
// powershell. Completion covers the subcommand (the big win for 70+ commands)
// plus the near-universal --json / --help flags.
func CompletionScript(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		return bashCompletion(), nil
	case "zsh":
		return zshCompletion(), nil
	case "fish":
		return fishCompletion(), nil
	case "powershell", "pwsh":
		return powershellCompletion(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (want bash, zsh, fish, or powershell)", shell)
	}
}

func runCompletion(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintf(stderr, "completion needs a shell: bash | zsh | fish | powershell\n")
		return 2
	}
	script, err := CompletionScript(args[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprint(stdout, script)
	return 0
}

func bashCompletion() string {
	var b strings.Builder
	b.WriteString("# meridian-cli bash completion. Install: meridian-cli completion bash > /usr/local/etc/bash_completion.d/meridian-cli\n")
	b.WriteString("_meridian_cli() {\n")
	b.WriteString("  local cur prev\n")
	b.WriteString("  cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	b.WriteString("  if [ \"$COMP_CWORD\" -eq 1 ]; then\n")
	fmt.Fprintf(&b, "    COMPREPLY=( $(compgen -W %q -- \"$cur\") )\n", strings.Join(names(), " "))
	b.WriteString("    return 0\n")
	b.WriteString("  fi\n")
	b.WriteString("  COMPREPLY=( $(compgen -W \"--json --help\" -- \"$cur\") )\n")
	b.WriteString("}\n")
	b.WriteString("complete -F _meridian_cli meridian-cli\n")
	return b.String()
}

func zshCompletion() string {
	var b strings.Builder
	b.WriteString("#compdef meridian-cli\n")
	b.WriteString("# meridian-cli zsh completion. Install: meridian-cli completion zsh > \"${fpath[1]}/_meridian-cli\"\n")
	b.WriteString("_meridian_cli() {\n")
	b.WriteString("  local -a cmds\n")
	b.WriteString("  cmds=(\n")
	for _, c := range commandList() {
		fmt.Fprintf(&b, "    %s\n", zshQuote(c.Name+":"+c.Desc))
	}
	b.WriteString("  )\n")
	b.WriteString("  if (( CURRENT == 2 )); then\n")
	b.WriteString("    _describe -t commands 'meridian-cli command' cmds\n")
	b.WriteString("  else\n")
	b.WriteString("    _values 'flag' '--json[output JSON]' '--help[show help]'\n")
	b.WriteString("  fi\n")
	b.WriteString("}\n")
	b.WriteString("_meridian_cli \"$@\"\n")
	return b.String()
}

func fishCompletion() string {
	var b strings.Builder
	b.WriteString("# meridian-cli fish completion. Install: meridian-cli completion fish > ~/.config/fish/completions/meridian-cli.fish\n")
	b.WriteString("complete -c meridian-cli -f\n")
	for _, c := range commandList() {
		fmt.Fprintf(&b, "complete -c meridian-cli -n __fish_use_subcommand -a %s -d %s\n", fishQuote(c.Name), fishQuote(c.Desc))
	}
	b.WriteString("complete -c meridian-cli -l json -d 'output JSON'\n")
	return b.String()
}

func powershellCompletion() string {
	var b strings.Builder
	b.WriteString("# meridian-cli PowerShell completion. Install: meridian-cli completion powershell | Out-String | Invoke-Expression  (add to $PROFILE)\n")
	b.WriteString("Register-ArgumentCompleter -CommandName meridian-cli -Native -ScriptBlock {\n")
	b.WriteString("  param($wordToComplete, $commandAst, $cursorPosition)\n")
	fmt.Fprintf(&b, "  @(%s) | Where-Object { $_ -like \"$wordToComplete*\" } | ForEach-Object {\n", psList(names()))
	b.WriteString("    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

// zshQuote single-quotes a value for a zsh array element, escaping embedded
// single quotes and the colon that separates value from description.
func zshQuote(s string) string {
	s = strings.ReplaceAll(s, "'", `'\''`)
	// escape every colon after the first (the first splits name:desc).
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i+1] + strings.ReplaceAll(s[i+1:], ":", `\:`)
	}
	return "'" + s + "'"
}

func fishQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `\'`) + "'" }

func psList(items []string) string {
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return strings.Join(q, ",")
}
