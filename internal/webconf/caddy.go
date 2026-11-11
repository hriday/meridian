package webconf

import (
	"fmt"
	"strconv"
	"strings"
)

// analyzeCaddy reads a Caddyfile and fills one Analysis. A Caddyfile is a set
// of site blocks (`address… { … }`), optionally preceded by a global options
// block (`{ … }`) and snippets (`(name) { … }`). Directives are newline-
// terminated; some carry their own `{ … }` body. Only directives that feed the
// model are read, so basic_auth hashes and other secrets in unread blocks are
// never surfaced. Caddy auto-HTTPS means a site with no `tls` is normal (not a
// missing-cert finding — handled in analyzeFindings).
func analyzeCaddy(configText string) (Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return Analysis{}, fmt.Errorf("caddy: no config provided")
	}
	a := newAnalysis()
	a.ServerType = "caddy"

	var stack []string // block kinds, innermost last: global | snippet | site | header | other
	var cur *Server
	hasImport := false

	for _, ln := range caddyLogicalLines(configText) {
		if ln == "}" {
			if len(stack) == 0 {
				continue
			}
			k := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if k == "site" && cur != nil {
				a.Servers = append(a.Servers, *cur)
				cur = nil
			}
			continue
		}
		opens := strings.HasSuffix(ln, "{")
		head := ln
		if opens {
			head = strings.TrimSpace(strings.TrimSuffix(ln, "{"))
		}
		toks := splitQuoted(head)

		if len(stack) == 0 { // top level
			if !opens {
				continue // stray top-level directive
			}
			switch {
			case head == "":
				stack = append(stack, "global")
			case strings.HasPrefix(head, "("):
				stack = append(stack, "snippet")
			default:
				cur = &Server{}
				for _, addr := range caddyAddresses(head) {
					cur.Names = append(cur.Names, caddyHost(addr))
					cur.Listeners = append(cur.Listeners, caddyListener(addr))
				}
				stack = append(stack, "site")
			}
			continue
		}

		switch stack[len(stack)-1] {
		case "site":
			if opens {
				name := strings.ToLower(first(toks))
				switch name {
				case "header":
					stack = append(stack, "header")
				case "reverse_proxy":
					caddyReverseProxy(cur, toks[1:])
					stack = append(stack, "other")
				case "tls":
					caddyTLS(cur, toks[1:])
					stack = append(stack, "other")
				default:
					stack = append(stack, "other")
				}
			} else {
				caddySiteDirective(cur, toks, &hasImport)
			}
		case "header":
			if opens {
				stack = append(stack, "other")
			} else if n := strings.TrimLeft(first(toks), "+-?"); n != "" {
				cur.Headers = append(cur.Headers, n)
			}
		default: // global, snippet, other
			if opens {
				stack = append(stack, "other")
			}
			if strings.EqualFold(first(toks), "import") {
				hasImport = true
			}
		}
	}

	if hasImport {
		a.Findings = append(a.Findings, Finding{
			Severity: "info", Category: "best-practice", Title: "Unresolved import",
			Detail: "this is a single Caddyfile — import directives couldn't be resolved", File: "pasted config",
		})
	}
	analyzeFindings(&a)
	return a, nil
}

// caddySiteDirective applies one simple (non-block) site directive.
func caddySiteDirective(cur *Server, toks []string, hasImport *bool) {
	if cur == nil || len(toks) == 0 {
		return
	}
	name, args := strings.ToLower(toks[0]), toks[1:]
	switch name {
	case "root":
		cur.Root = caddyRoot(args)
	case "reverse_proxy":
		caddyReverseProxy(cur, args)
	case "file_server":
		if containsFold(args, "browse") {
			cur.Locations = append(cur.Locations, Location{Path: "/", Kind: "static", Flags: []string{"autoindex on"}})
		}
	case "tls":
		caddyTLS(cur, args)
	case "header":
		if len(args) >= 1 {
			if n := strings.TrimLeft(args[0], "+-?"); n != "" {
				cur.Headers = append(cur.Headers, n)
			}
		}
	case "redir":
		loc := Location{Kind: "redirect"}
		if len(args) >= 1 {
			loc.Path = args[0]
		}
		if len(args) >= 2 {
			loc.Target = args[1]
		}
		cur.Locations = append(cur.Locations, loc)
	case "import":
		*hasImport = true
	}
}

// caddyReverseProxy records a reverse_proxy as a proxy Location (path optional,
// first backend as the target).
func caddyReverseProxy(cur *Server, args []string) {
	if cur == nil {
		return
	}
	path, i := "/", 0
	if len(args) > 0 && (strings.HasPrefix(args[0], "/") || strings.HasPrefix(args[0], "@") || args[0] == "*") {
		path, i = args[0], 1
	}
	target := ""
	if i < len(args) {
		target = args[i]
	}
	cur.Locations = append(cur.Locations, Location{Path: path, Kind: "proxy", Target: target})
}

// caddyTLS sets TLSCert for the explicit cert-file form; the email / internal /
// automatic forms leave it empty (Caddy provisions a cert).
func caddyTLS(cur *Server, args []string) {
	if cur == nil || len(args) == 0 {
		return
	}
	a0 := args[0]
	if strings.Contains(a0, "/") || strings.HasSuffix(a0, ".pem") || strings.HasSuffix(a0, ".crt") {
		cur.TLSCert = a0
	}
}

// caddyLogicalLines trims lines, drops blanks and full-line comments.
func caddyLogicalLines(text string) []string {
	var out []string
	for _, raw := range strings.Split(text, "\n") {
		s := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		out = append(out, s)
	}
	return out
}

// caddyAddresses splits a site header ("a.com, b.com") into addresses.
func caddyAddresses(head string) []string {
	return strings.Fields(strings.ReplaceAll(head, ",", " "))
}

// caddyHost strips scheme and port from an address for the display name.
func caddyHost(addr string) string {
	addr = strings.TrimPrefix(strings.TrimPrefix(addr, "https://"), "http://")
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		if _, err := strconv.Atoi(addr[i+1:]); err == nil {
			addr = addr[:i]
		}
	}
	return addr
}

// caddyListener maps an address to a Listener; a bare hostname or https:// is
// auto-HTTPS on 443.
func caddyListener(addr string) Listener {
	scheme := ""
	switch {
	case strings.HasPrefix(addr, "https://"):
		scheme, addr = "https", addr[len("https://"):]
	case strings.HasPrefix(addr, "http://"):
		scheme, addr = "http", addr[len("http://"):]
	}
	host, port := addr, 0
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		if p, err := strconv.Atoi(addr[i+1:]); err == nil {
			host, port = addr[:i], p
		}
	}
	ln := Listener{Addr: host}
	switch {
	case scheme == "http":
		ln.Port = orDefault(port, 80)
	case scheme == "https":
		ln.Port, ln.SSL = orDefault(port, 443), true
	case port == 80:
		ln.Port = 80
	case port != 0:
		ln.Port, ln.SSL = port, port == 443
	default: // bare hostname → auto-HTTPS
		ln.Port, ln.SSL = 443, true
	}
	return ln
}

// caddyRoot returns the filesystem path of a `root [matcher] <path>` directive.
func caddyRoot(args []string) string {
	for _, a := range args {
		if a != "*" && !strings.HasPrefix(a, "@") {
			return a
		}
	}
	return first(args)
}

func containsFold(xs []string, want string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, want) {
			return true
		}
	}
	return false
}

func orDefault(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}
