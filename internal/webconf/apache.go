package webconf

import (
	"fmt"
	"strconv"
	"strings"
)

// analyzeApache reads an Apache httpd config and fills one Analysis. Apache is
// line-oriented with case-insensitive directives and <Section …>…</Section>
// containers; <VirtualHost> blocks become Servers. Only directives that feed the
// model are read, so inline secrets (AuthLDAPBindPassword, SSL key material)
// sit in unread directives and are never surfaced. Includes can't be resolved
// from a single file, so they're noted.
func analyzeApache(configText string) (Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return Analysis{}, fmt.Errorf("apache: no config provided")
	}
	a := newAnalysis()
	a.ServerType = "apache"

	globalTokens := ""
	hasInclude := false
	var cur *Server // current <VirtualHost>
	// current inner <Directory>/<Location> block
	var innerActive bool
	var innerKind, innerPath string
	var innerIndexes bool

	flushInner := func() {
		if innerActive && cur != nil {
			if innerKind == "location" {
				loc := Location{Path: innerPath, Kind: "static"}
				if innerIndexes {
					loc.Flags = append(loc.Flags, "autoindex on")
				}
				cur.Locations = append(cur.Locations, loc)
			} else if innerIndexes { // <Directory> with Options Indexes
				cur.Locations = append(cur.Locations, Location{Path: innerPath, Kind: "static", Flags: []string{"autoindex on"}})
			}
		}
		innerActive, innerKind, innerPath, innerIndexes = false, "", "", false
	}
	closeVHost := func() {
		flushInner()
		if cur != nil {
			if cur.ServerTokens == "" {
				cur.ServerTokens = globalTokens
			}
			a.Servers = append(a.Servers, *cur)
			cur = nil
		}
	}

	for _, ln := range apacheLogicalLines(configText) {
		switch {
		case strings.HasPrefix(ln, "</"):
			name := strings.ToLower(strings.Trim(ln, "</> \t"))
			if name == "virtualhost" {
				closeVHost()
			} else if name == "directory" || name == "location" || name == "directorymatch" || name == "locationmatch" || name == "files" || name == "filesmatch" {
				flushInner()
			}
		case strings.HasPrefix(ln, "<"):
			name, args := apacheSection(ln)
			switch strings.ToLower(name) {
			case "virtualhost":
				closeVHost()
				cur = &Server{}
				for _, addr := range args {
					cur.Listeners = append(cur.Listeners, apacheListener(addr))
				}
			case "location", "locationmatch":
				flushInner()
				innerActive, innerKind = true, "location"
				if len(args) > 0 {
					innerPath = args[0]
				}
			case "directory", "directorymatch":
				flushInner()
				innerActive, innerKind = true, "directory"
				if len(args) > 0 {
					innerPath = args[0]
				}
			}
		default:
			name, args := apacheDirective(ln)
			lname := strings.ToLower(name)
			switch lname {
			case "servertokens":
				globalTokens = first(args)
				if cur != nil {
					cur.ServerTokens = first(args)
				}
			case "include", "includeoptional":
				hasInclude = true
			case "options":
				if innerActive && apacheHasIndexes(args) {
					innerIndexes = true
				}
			}
			if cur == nil {
				continue
			}
			switch lname {
			case "servername", "serveralias":
				cur.Names = append(cur.Names, args...)
			case "documentroot":
				cur.Root = first(args)
			case "sslengine":
				if strings.EqualFold(first(args), "on") {
					for i := range cur.Listeners {
						cur.Listeners[i].SSL = true
					}
				}
			case "sslprotocol":
				cur.TLSProtocols = apacheSSLProtocols(args)
			case "sslcertificatefile":
				cur.TLSCert = first(args)
			case "header":
				if h := apacheHeaderName(args); h != "" {
					cur.Headers = append(cur.Headers, h)
				}
			case "proxypass":
				// ProxyPass <path> <url> [opts]; skip the "!" exclusion form
				if len(args) >= 2 && args[1] != "!" {
					cur.Locations = append(cur.Locations, Location{Path: args[0], Kind: "proxy", Target: args[1]})
				}
			case "redirect", "redirectmatch":
				cur.Locations = append(cur.Locations, apacheRedirect(args))
			}
		}
	}
	closeVHost() // flush an unclosed final vhost

	if hasInclude {
		a.Findings = append(a.Findings, Finding{
			Severity: "info", Category: "best-practice", Title: "Unresolved include",
			Detail: "this is a single file, not a full server dump — Include/IncludeOptional directives couldn't be resolved", File: "pasted config",
		})
	}
	analyzeFindings(&a)
	return a, nil
}

// apacheLogicalLines splits the config into trimmed logical lines, joining
// backslash line-continuations and dropping blanks and '#' comments.
func apacheLogicalLines(text string) []string {
	var out []string
	var buf strings.Builder
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.HasSuffix(line, "\\") {
			buf.WriteString(strings.TrimSuffix(line, "\\"))
			buf.WriteByte(' ')
			continue
		}
		buf.WriteString(line)
		s := strings.TrimSpace(buf.String())
		buf.Reset()
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		out = append(out, s)
	}
	if s := strings.TrimSpace(buf.String()); s != "" && !strings.HasPrefix(s, "#") {
		out = append(out, s)
	}
	return out
}

// apacheSection parses "<Name arg1 arg2>" into the name and its args.
func apacheSection(ln string) (string, []string) {
	inner := strings.TrimSuffix(strings.TrimPrefix(ln, "<"), ">")
	inner = strings.TrimSpace(strings.TrimSuffix(inner, ">"))
	fields := splitQuoted(inner)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

// apacheDirective parses "Name arg1 arg2" into the name and its args.
func apacheDirective(ln string) (string, []string) {
	fields := splitQuoted(ln)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], fields[1:]
}

// apacheListener parses a VirtualHost address spec (*:443, _default_:80, 443,
// [::]:443) into a Listener; port 443 implies SSL.
func apacheListener(spec string) Listener {
	ln := Listener{}
	host, portStr := spec, ""
	if i := strings.LastIndex(spec, ":"); i >= 0 {
		host, portStr = spec[:i], spec[i+1:]
	}
	if p, err := strconv.Atoi(portStr); err == nil {
		ln.Addr, ln.Port = host, p
	} else if p, err := strconv.Atoi(spec); err == nil {
		ln.Port = p
	} else {
		ln.Addr = spec
	}
	if ln.Port == 443 {
		ln.SSL = true
	}
	return ln
}

// apacheSSLProtocols resolves an SSLProtocol directive (all -SSLv3 -TLSv1 /
// TLSv1.2 TLSv1 / +TLSv1.3) into the enabled protocol list.
func apacheSSLProtocols(args []string) []string {
	order := []string{"SSLv3", "TLSv1", "TLSv1.1", "TLSv1.2", "TLSv1.3"}
	enabled := map[string]bool{}
	for _, tok := range args {
		switch {
		case strings.EqualFold(tok, "all"):
			for _, p := range order {
				enabled[p] = true
			}
		case strings.HasPrefix(tok, "-"):
			delete(enabled, canonTLS(tok[1:]))
		case strings.HasPrefix(tok, "+"):
			enabled[canonTLS(tok[1:])] = true
		default:
			enabled[canonTLS(tok)] = true
		}
	}
	var out []string
	for _, p := range order {
		if enabled[p] {
			out = append(out, p)
		}
	}
	return out
}

func canonTLS(s string) string {
	switch strings.ToLower(s) {
	case "sslv3":
		return "SSLv3"
	case "tlsv1":
		return "TLSv1"
	case "tlsv1.1":
		return "TLSv1.1"
	case "tlsv1.2":
		return "TLSv1.2"
	case "tlsv1.3":
		return "TLSv1.3"
	}
	return s
}

// apacheHeaderName returns the header name a `Header` directive sets, or "" for
// unset/other actions. Form: Header [condition] <action> <Name> …
func apacheHeaderName(args []string) string {
	i := 0
	for i < len(args) && (strings.EqualFold(args[i], "always") || strings.EqualFold(args[i], "early")) {
		i++
	}
	if i >= len(args) {
		return ""
	}
	switch strings.ToLower(args[i]) {
	case "set", "setifempty", "append", "add", "merge", "edit", "edit*":
		if i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// apacheRedirect parses "Redirect [status] <url-path> <url>" into a Location.
func apacheRedirect(args []string) Location {
	a := args
	if len(a) > 0 && !strings.HasPrefix(a[0], "/") { // drop a leading status word
		a = a[1:]
	}
	loc := Location{Kind: "redirect"}
	if len(a) > 0 {
		loc.Path = a[0]
	}
	if len(a) > 1 {
		loc.Target = a[1]
	}
	return loc
}

func apacheHasIndexes(args []string) bool {
	for _, a := range args {
		if strings.EqualFold(a, "Indexes") || strings.EqualFold(a, "+Indexes") {
			return true
		}
	}
	return false
}

// first returns the first element or "".
func first(xs []string) string {
	if len(xs) > 0 {
		return xs[0]
	}
	return ""
}

// splitQuoted splits a line on whitespace, honoring double-quoted substrings and
// stripping the surrounding quotes.
func splitQuoted(s string) []string {
	var out []string
	var b strings.Builder
	inQuote, has := false, false
	flush := func() {
		if has {
			out = append(out, b.String())
			b.Reset()
			has = false
		}
	}
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			has = true
		case (r == ' ' || r == '\t') && !inQuote:
			flush()
		default:
			b.WriteRune(r)
			has = true
		}
	}
	flush()
	return out
}
