package webconf

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func with(a Analysis, substr string) []Finding {
	var out []Finding
	for _, f := range a.Findings {
		if strings.Contains(strings.ToLower(f.Title+" "+f.Detail), strings.ToLower(substr)) {
			out = append(out, f)
		}
	}
	return out
}

func TestFindingUndefinedUpstream(t *testing.T) {
	a := extractCfg(t, `http {
    upstream good { server 1.1.1.1:80; }
    server {
        location /a { proxy_pass http://good; }
        location /b { proxy_pass http://missing; }
        location /c { proxy_pass http://10.0.0.1:8080; }
        location /d { proxy_pass http://$backend; }
    }
}`)
	f := with(a, "missing")
	require.Len(t, f, 1)
	require.Equal(t, "error", f[0].Severity)
	require.Empty(t, with(a, "good"))     // defined upstream: no finding
	require.Empty(t, with(a, "10.0.0.1")) // host:port: no finding
}

func TestFindingDuplicateServerNameSamePort(t *testing.T) {
	a := extractCfg(t, `http {
    server { listen 443 ssl; ssl_certificate x; server_name a.com; }
    server { listen 443 ssl; ssl_certificate x; server_name a.com; }
    server { listen 80; server_name b.com; }
    server { listen 443 ssl; ssl_certificate x; server_name b.com; }
}`)
	dup := with(a, "server_name")
	require.Len(t, dup, 1) // a.com twice on :443
	require.Contains(t, dup[0].Title+dup[0].Detail, "a.com")
}

func TestFindingSslListenNoCert(t *testing.T) {
	a := extractCfg(t, `http { server { listen 8443 ssl; server_name x; } }`)
	require.NotEmpty(t, with(a, "certificate"))
	require.Equal(t, "error", with(a, "certificate")[0].Severity)
}

func TestFindingDeprecatedTLSAndAutoindex(t *testing.T) {
	a := extractCfg(t, `http {
    server {
        listen 443 ssl; ssl_certificate x; server_name s;
        ssl_protocols TLSv1 TLSv1.2;
        add_header Strict-Transport-Security "max-age=1";
        location /u { autoindex on; }
    }
}`)
	require.NotEmpty(t, with(a, "TLSv1"))
	require.Equal(t, "security", with(a, "TLSv1")[0].Category)
	require.NotEmpty(t, with(a, "autoindex"))
}

func TestFindingServerTokensInheritanceOff(t *testing.T) {
	a := extractCfg(t, `http { server_tokens off; server { listen 80; server_name x; } }`)
	require.Empty(t, with(a, "server_tokens"))
}

func TestFindingServerTokensMissing(t *testing.T) {
	a := extractCfg(t, `http { server { listen 80; server_name x; } }`)
	require.NotEmpty(t, with(a, "server_tokens"))
	require.Equal(t, "best-practice", with(a, "server_tokens")[0].Category)
}

func TestFindingHSTSInheritanceCovered(t *testing.T) {
	a := extractCfg(t, `http {
    add_header Strict-Transport-Security "max-age=1";
    server { listen 443 ssl; ssl_certificate x; server_name s; }
}`)
	require.Empty(t, with(a, "HSTS"))
}

func TestFindingHSTSMissing(t *testing.T) {
	a := extractCfg(t, `http {
    server { listen 443 ssl; ssl_certificate x; server_name s; }
}`)
	require.NotEmpty(t, with(a, "HSTS"))
}

func TestRoutingUsedBy(t *testing.T) {
	a := extractCfg(t, `http {
    upstream api { server 1.1.1.1:80; }
    server { server_name site.com; location /api { proxy_pass http://api; } }
}`)
	require.Len(t, a.Upstreams, 1)
	require.Len(t, a.Upstreams[0].UsedBy, 1)
	require.Contains(t, a.Upstreams[0].UsedBy[0], "/api")
}

func TestCounts(t *testing.T) {
	a := extractCfg(t, `http {
    upstream api { server 1.1.1.1:80; }
    server {
        listen 80; listen 443 ssl; ssl_certificate x; server_name c.com;
        server_tokens off;
        add_header Strict-Transport-Security "m";
        add_header X-Content-Type-Options nosniff;
        add_header X-Frame-Options SAMEORIGIN;
        location / { try_files $uri =404; }
        location /api { proxy_pass http://api; }
    }
}`)
	require.Equal(t, 1, a.Counts.Servers)
	require.Equal(t, 2, a.Counts.Listeners)
	require.Equal(t, 1, a.Counts.Upstreams)
	require.Equal(t, 2, a.Counts.Locations)
	require.Equal(t, 0, a.Counts.Errors)
	require.Equal(t, 0, a.Counts.Warnings)
}
