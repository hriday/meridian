package webconf

import (
	"strconv"
	"strings"
)

// extract walks the parsed directives and builds the Analysis model: server
// blocks, listeners, locations, upstreams and TLS. nginx Plus directives are
// read through without special handling.
func extract(dirs []*Directive) Analysis {
	a := newAnalysis()
	for _, http := range findBlocks(dirs, "http") {
		// http-level inheritance context
		if st := lastArg(http, "server_tokens"); st != "" {
			a.httpTokens = st
		}
		for _, h := range http.children("add_header") {
			a.httpHeaders = append(a.httpHeaders, h.arg(0))
		}
		for _, s := range http.children("server") {
			a.Servers = append(a.Servers, readServer(s))
		}
		for _, u := range http.children("upstream") {
			a.Upstreams = append(a.Upstreams, readUpstream(u))
		}
	}
	return a
}

// findBlocks returns directives named name among dirs (top level); nginx config
// wraps servers/upstreams in `http`, but we tolerate them at top level too.
func findBlocks(dirs []*Directive, name string) []*Directive {
	var out []*Directive
	for _, d := range dirs {
		if d.Name == name {
			out = append(out, d)
		}
	}
	if len(out) == 0 && name == "http" {
		// no http wrapper: treat the top level as the http context.
		return []*Directive{{Block: dirs}}
	}
	return out
}

func readServer(s *Directive) Server {
	srv := Server{File: s.File, Line: s.Line}
	for _, sn := range s.children("server_name") {
		srv.Names = append(srv.Names, sn.Args...)
	}
	for _, l := range s.children("listen") {
		srv.Listeners = append(srv.Listeners, readListener(l))
	}
	srv.Root = lastArg(s, "root")
	if p := s.children("ssl_protocols"); len(p) > 0 {
		srv.TLSProtocols = p[0].Args
	}
	srv.TLSCert = lastArg(s, "ssl_certificate")
	srv.ServerTokens = lastArg(s, "server_tokens")
	for _, al := range s.children("allow") {
		srv.Access = append(srv.Access, "allow "+al.arg(0))
	}
	for _, dn := range s.children("deny") {
		srv.Access = append(srv.Access, "deny "+dn.arg(0))
	}
	for _, h := range s.children("add_header") {
		srv.Headers = append(srv.Headers, h.arg(0))
	}
	for _, loc := range s.children("location") {
		srv.Locations = append(srv.Locations, readLocation(loc))
	}
	return srv
}

func readListener(l *Directive) Listener {
	spec := l.arg(0)
	ln := Listener{}
	if i := strings.LastIndex(spec, ":"); i >= 0 {
		ln.Addr = spec[:i]
		ln.Port, _ = strconv.Atoi(spec[i+1:])
	} else if n, err := strconv.Atoi(spec); err == nil {
		ln.Port = n
	} else {
		ln.Addr = spec
	}
	for _, opt := range l.Args {
		switch opt {
		case "ssl":
			ln.SSL = true
		case "http2":
			ln.HTTP2 = true
		case "default_server":
			ln.Default = true
		}
	}
	return ln
}

func readLocation(loc *Directive) Location {
	// path is the last positional arg (after an optional modifier like = / ~).
	path := loc.arg(0)
	if len(loc.Args) > 1 {
		path = loc.Args[len(loc.Args)-1]
	}
	out := Location{Path: path, Kind: "static"}
	if pp := loc.children("proxy_pass"); len(pp) > 0 {
		out.Kind = "proxy"
		out.Target = upstreamName(pp[0].arg(0))
	} else if r := loc.children("return"); len(r) > 0 {
		out.Kind = "redirect"
		out.Target = strings.Join(r[0].Args, " ")
	} else if al := loc.children("alias"); len(al) > 0 {
		out.Kind = "alias"
		out.Target = al[0].arg(0)
	}
	if ai := lastArg(loc, "autoindex"); ai == "on" {
		out.Flags = append(out.Flags, "autoindex on")
	}
	return out
}

func readUpstream(u *Directive) Upstream {
	up := Upstream{Name: u.arg(0), File: u.File, Line: u.Line}
	for _, srv := range u.children("server") {
		up.Servers = append(up.Servers, srv.arg(0))
	}
	for _, policy := range []string{"least_conn", "ip_hash", "random", "hash", "least_time"} {
		if len(u.children(policy)) > 0 {
			up.Policy = policy
			break
		}
	}
	return up
}

// upstreamName returns the bare upstream name a proxy_pass target names, or ""
// when the target has a scheme host/port or a variable (not an upstream ref).
func upstreamName(target string) string {
	t := strings.TrimPrefix(strings.TrimPrefix(target, "https://"), "http://")
	if t == target { // no http(s):// scheme
		return ""
	}
	// strip a trailing path
	if i := strings.IndexByte(t, '/'); i >= 0 {
		t = t[:i]
	}
	if t == "" || strings.ContainsAny(t, "$:") {
		return "" // variable, or host:port — not an upstream name
	}
	return t
}

// lastArg returns the first arg of the last directive named name, or "".
func lastArg(d *Directive, name string) string {
	c := d.children(name)
	if len(c) == 0 {
		return ""
	}
	return c[len(c)-1].arg(0)
}
