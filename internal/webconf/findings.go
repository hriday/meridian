package webconf

import (
	"fmt"
	"sort"
	"strings"
)

// analyzeFindings fills the upstream-usage map, the findings list, and the
// counts. Called by extract after the model is built.
func analyzeFindings(a *Analysis) {
	upIndex := map[string]int{}
	for i, u := range a.Upstreams {
		upIndex[u.Name] = i
	}
	add := func(sev, cat, title, detail, file string, line int) {
		a.Findings = append(a.Findings, Finding{Severity: sev, Category: cat, Title: title, Detail: detail, File: file, Line: line})
	}

	// Routing + undefined upstream.
	for si := range a.Servers {
		s := &a.Servers[si]
		disp := serverDisplay(s)
		for _, loc := range s.Locations {
			if loc.Kind != "proxy" || loc.Target == "" {
				continue
			}
			if idx, ok := upIndex[loc.Target]; ok {
				a.Upstreams[idx].UsedBy = append(a.Upstreams[idx].UsedBy, disp+" "+loc.Path)
			} else {
				add("error", "breaking",
					"proxy_pass to undefined upstream "+loc.Target,
					fmt.Sprintf("%s %s proxies to upstream %q, which is never defined", disp, loc.Path, loc.Target),
					s.File, s.Line)
			}
		}
	}

	// Duplicate server_name on the same listen host:port (ignoring the "_"
	// catch-all).
	type nk struct{ name, hostport string }
	occ := map[nk][]int{}
	for si := range a.Servers {
		s := &a.Servers[si]
		for _, name := range s.Names {
			if name == "" || name == "_" {
				continue
			}
			for _, l := range s.Listeners {
				occ[nk{name, fmt.Sprintf("%s:%d", l.Addr, l.Port)}] = append(occ[nk{name, fmt.Sprintf("%s:%d", l.Addr, l.Port)}], si)
			}
		}
	}
	for key, servers := range occ {
		if len(distinct(servers)) > 1 {
			s := &a.Servers[servers[0]]
			add("error", "breaking",
				fmt.Sprintf("Duplicate server_name %q on %s", key.name, key.hostport),
				"the same name is defined on the same listen address more than once; the later block is shadowed",
				s.File, s.Line)
		}
	}

	for si := range a.Servers {
		s := &a.Servers[si]
		disp := serverDisplay(s)
		ssl := false
		for _, l := range s.Listeners {
			if l.SSL {
				ssl = true
			}
		}
		if ssl && s.TLSCert == "" {
			add("error", "breaking", "listen ssl with no ssl_certificate",
				disp+" has an SSL listener but no ssl_certificate — the TLS listener can't start", s.File, s.Line)
		}
		for _, p := range s.TLSProtocols {
			if p == "TLSv1" || p == "TLSv1.1" {
				add("warning", "security", "Deprecated TLS version "+p,
					disp+" enables "+p+" in ssl_protocols", s.File, s.Line)
			}
		}
		for _, loc := range s.Locations {
			for _, fl := range loc.Flags {
				if fl == "autoindex on" {
					add("warning", "security", "autoindex on exposes a directory listing",
						disp+" "+loc.Path+" has autoindex on", s.File, s.Line)
				}
			}
		}
		if ssl {
			if !effectiveHeader(s, a, "Strict-Transport-Security") {
				add("warning", "security", "Missing HSTS header",
					disp+" serves TLS but sets no Strict-Transport-Security (HSTS) header", s.File, s.Line)
			}
			for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options"} {
				if !effectiveHeader(s, a, h) {
					add("warning", "security", "Missing "+h+" header",
						disp+" does not set "+h, s.File, s.Line)
				}
			}
		}
		if len(s.Names) == 0 {
			add("warning", "best-practice", "Server block with no server_name",
				fmt.Sprintf("the server block at %s:%d has no server_name", s.File, s.Line), s.File, s.Line)
		}
	}

	if serverTokensLeak(a) {
		add("warning", "best-practice", "server_tokens not off — leaks the nginx version",
			"set server_tokens off; at http or server level so the version doesn't appear in responses", "", 0)
	}

	listeners, locations, errs, warns := 0, 0, 0, 0
	for _, s := range a.Servers {
		listeners += len(s.Listeners)
		locations += len(s.Locations)
	}
	for _, f := range a.Findings {
		switch f.Severity {
		case "error":
			errs++
		case "warning":
			warns++
		}
	}
	a.Counts = Counts{Servers: len(a.Servers), Listeners: listeners, Upstreams: len(a.Upstreams),
		Locations: locations, Errors: errs, Warnings: warns}

	rank := map[string]int{"error": 0, "warning": 1, "info": 2}
	sort.SliceStable(a.Findings, func(i, j int) bool {
		fi, fj := a.Findings[i], a.Findings[j]
		if rank[fi.Severity] != rank[fj.Severity] {
			return rank[fi.Severity] < rank[fj.Severity]
		}
		if fi.File != fj.File {
			return fi.File < fj.File
		}
		return fi.Line < fj.Line
	})
}

func serverDisplay(s *Server) string {
	if len(s.Names) > 0 {
		return s.Names[0]
	}
	return "(unnamed server)"
}

func distinct(xs []int) []int {
	seen := map[int]bool{}
	var out []int
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// effectiveHeader reports whether the header is set at the server level or
// inherited from http.
func effectiveHeader(s *Server, a *Analysis, name string) bool {
	for _, h := range s.Headers {
		if strings.EqualFold(h, name) {
			return true
		}
	}
	for _, h := range a.httpHeaders {
		if strings.EqualFold(h, name) {
			return true
		}
	}
	return false
}

// serverTokensLeak reports whether server_tokens is left non-off anywhere it
// matters, accounting for http→server inheritance.
func serverTokensLeak(a *Analysis) bool {
	if a.httpTokens == "off" {
		return false
	}
	if len(a.Servers) == 0 {
		return a.httpTokens != "off"
	}
	for _, s := range a.Servers {
		eff := s.ServerTokens
		if eff == "" {
			eff = a.httpTokens
		}
		if eff != "off" {
			return true
		}
	}
	return false
}
