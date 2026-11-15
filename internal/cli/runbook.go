package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"strings"
	"time"
)

func runRunbook(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("runbook", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "md", "report format: md, json, or html")
	asJSON := fs.Bool("json", false, "shorthand for --format json")
	outFile := fs.String("out", "", "write the report to a file instead of stdout")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) == 1 && pos[0] == "list" {
		for _, p := range Runbooks() {
			fmt.Fprintf(stdout, "%-14s %s (%d checks)\n", p.ID, p.Desc, p.Steps)
		}
		return 0
	}
	if len(pos) != 2 {
		fmt.Fprintf(stderr, "runbook needs a profile and a target (or `runbook list`)\n%s", usage)
		return 2
	}

	f := *format
	if *asJSON {
		f = "json"
	}
	fmt.Fprintf(stderr, "Running runbook %q against %s…\n", pos[0], pos[1])
	report, code := RunbookReport(pos[0], pos[1], f)
	if code == 2 {
		fmt.Fprintln(stderr, report)
		return 2
	}
	if *outFile != "" {
		if err := os.WriteFile(*outFile, []byte(report+"\n"), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stderr, "wrote %s\n", *outFile)
		return code
	}
	fmt.Fprintln(stdout, report)
	return code
}

// A runbook is a named composite profile: a sequence of single-target tools run
// against one target, assembled into one report. It reuses Run per step, so a
// profile is just a list of commands — no per-tool orchestration.

type rbStep struct {
	cmd  string
	args []string // "{t}" is replaced by the target
}

type rbProfile struct {
	id, name, desc string
	steps          []rbStep
}

var rbProfiles = []rbProfile{
	{
		id: "audit-domain", name: "Audit Domain",
		desc: "Registration, DNS posture, email auth, subdomains, TLS ciphers and HTTP security headers for a domain.",
		steps: []rbStep{
			{"lookup", []string{"{t}"}},
			{"dnsaudit", []string{"{t}"}},
			{"emailauth", []string{"{t}"}},
			{"subdomains", []string{"{t}"}},
			{"cipherscan", []string{"{t}"}},
			{"sechdr", []string{"https://{t}"}},
		},
	},
	{
		id: "triage-host", name: "Triage Host",
		desc: "Reachability, latency, path, TLS and common open ports for a host.",
		steps: []rbStep{
			{"ping", []string{"{t}", "--count", "3"}},
			{"tcpping", []string{"{t}", "443"}},
			{"trace", []string{"{t}"}},
			{"tls", []string{"{t}"}},
			{"portscan", []string{"{t}", "--common"}},
		},
	},
}

// RunbookInfo describes a profile (for listing in the CLI/GUI).
type RunbookInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Steps int    `json:"steps"`
}

// RunbookStep is one executed step's result.
type RunbookStep struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Output  string   `json:"output"`
	OK      bool     `json:"ok"`
}

// RunbookData is a whole executed runbook.
type RunbookData struct {
	Profile   string        `json:"profile"`
	Name      string        `json:"name"`
	Target    string        `json:"target"`
	Generated string        `json:"generated"`
	Steps     []RunbookStep `json:"steps"`
}

// Runbooks lists the available profiles.
func Runbooks() []RunbookInfo {
	out := make([]RunbookInfo, 0, len(rbProfiles))
	for _, p := range rbProfiles {
		out = append(out, RunbookInfo{ID: p.id, Name: p.name, Desc: p.desc, Steps: len(p.steps)})
	}
	return out
}

func findProfile(id string) (rbProfile, bool) {
	for _, p := range rbProfiles {
		if p.id == id {
			return p, true
		}
	}
	return rbProfile{}, false
}

// RunRunbookData runs a profile against target and returns the structured
// result. ok is false if the profile id is unknown.
func RunRunbookData(profileID, target string) (RunbookData, bool) {
	p, found := findProfile(profileID)
	if !found {
		return RunbookData{}, false
	}
	d := RunbookData{Profile: p.id, Name: p.name, Target: target, Generated: time.Now().UTC().Format(time.RFC3339)}
	for _, s := range p.steps {
		args := make([]string, len(s.args))
		for i, a := range s.args {
			args[i] = strings.ReplaceAll(a, "{t}", target)
		}
		var out, errb bytes.Buffer
		code := Run(append([]string{s.cmd}, args...), &out, &errb)
		o := strings.TrimRight(out.String(), "\n")
		if code != 0 && o == "" {
			o = strings.TrimSpace(errb.String())
		}
		d.Steps = append(d.Steps, RunbookStep{Command: s.cmd, Args: args, Output: o, OK: code == 0})
	}
	return d, true
}

// RunbookReport runs a profile and renders it in format ("md", "json", "html").
// code is 0 on success, 2 for an unknown profile.
func RunbookReport(profileID, target, format string) (string, int) {
	d, ok := RunRunbookData(profileID, target)
	if !ok {
		var ids []string
		for _, p := range rbProfiles {
			ids = append(ids, p.id)
		}
		return fmt.Sprintf("unknown runbook %q (have: %s)", profileID, strings.Join(ids, ", ")), 2
	}
	switch format {
	case "json":
		b, _ := json.Marshal(d)
		return string(b), 0
	case "html":
		return renderRunbookHTML(d), 0
	default:
		return renderRunbookMD(d), 0
	}
}

func renderRunbookMD(d RunbookData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Runbook: %s — %s\n\n", d.Name, d.Target)
	fmt.Fprintf(&b, "_%d checks · generated %s_\n", len(d.Steps), d.Generated)
	for _, s := range d.Steps {
		status := ""
		if !s.OK {
			status = " ⚠️"
		}
		fmt.Fprintf(&b, "\n## %s%s\n\n```\n%s\n```\n", s.Command, status, s.Output)
	}
	return b.String()
}

func renderRunbookHTML(d RunbookData) string {
	var b strings.Builder
	title := fmt.Sprintf("Runbook: %s — %s", d.Name, d.Target)
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title><style>body{font-family:system-ui,sans-serif;max-width:900px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}h2{border-bottom:1px solid #ddd;padding-bottom:4px;margin-top:1.5rem}pre{background:#0d1117;color:#c9d1d9;padding:12px;border-radius:8px;overflow:auto;font-size:13px;line-height:1.4}.warn{color:#c0392b}</style></head><body>")
	fmt.Fprintf(&b, "<h1>%s</h1><p>%d checks · generated %s</p>", html.EscapeString(title), len(d.Steps), html.EscapeString(d.Generated))
	for _, s := range d.Steps {
		warn := ""
		if !s.OK {
			warn = " <span class=\"warn\">⚠️</span>"
		}
		fmt.Fprintf(&b, "<h2>%s%s</h2><pre>%s</pre>", html.EscapeString(s.Command), warn, html.EscapeString(s.Output))
	}
	b.WriteString("</body></html>")
	return b.String()
}
