package webconf

import (
	"fmt"
	"strings"
)

// Analyze parses a web-server config (an `nginx -T` dump or a single file) and
// returns the structured analysis. server defaults to "nginx", the only server
// supported in v1. Processing is entirely offline.
func Analyze(configText, server string) (Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return Analysis{}, fmt.Errorf("no config provided — paste or load an `nginx -T` dump")
	}
	s := strings.ToLower(strings.TrimSpace(server))
	if s == "" {
		s = "nginx"
	}
	switch s {
	case "apache":
		return analyzeApache(configText)
	case "nginx":
		// handled below
	default:
		return Analysis{}, fmt.Errorf("server %q is not supported — use nginx, apache, or caddy", server)
	}

	dirs, err := parse(configText)
	if err != nil {
		return Analysis{}, err
	}
	a := extract(dirs)

	// Without `# configuration file` markers this is a single pasted file, not
	// an `nginx -T` dump, so any include is unresolved — say so plainly.
	if !strings.Contains(configText, markerPrefix) {
		if incs := collectIncludes(dirs); len(incs) > 0 {
			a.Findings = append(a.Findings, Finding{
				Severity: "info", Category: "best-practice",
				Title: "Unresolved include",
				Detail: fmt.Sprintf("this looks like a single file, not an `nginx -T` dump — %d include(s) couldn't be resolved: %s",
					len(incs), strings.Join(incs, ", ")),
				File: "pasted config",
			})
		}
	}
	return a, nil
}

func collectIncludes(dirs []*Directive) []string {
	var out []string
	var walk func([]*Directive)
	walk = func(ds []*Directive) {
		for _, d := range ds {
			if d.Name == "include" {
				out = append(out, d.arg(0))
			}
			walk(d.Block)
		}
	}
	walk(dirs)
	return out
}
