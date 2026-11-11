package webconf

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic Caddyfile: a global options block, an auto-HTTPS site with a
// reverse proxy, security headers and a basic_auth hash the parser must ignore;
// a site with an explicit cert + file_server browse; and a site using the tls
// email (automatic) form.
const caddyCfg = `{
    email admin@example.com
    admin off
}

example.com, www.example.com {
    root * /var/www/example
    encode gzip
    reverse_proxy /api/* http://10.0.0.5:8080
    header Strict-Transport-Security "max-age=31536000"
    header X-Content-Type-Options nosniff
    header X-Frame-Options SAMEORIGIN
    basic_auth {
        admin $2a$14$HASHSECRETxyz0000000000
    }
}

files.example.com {
    tls /etc/ssl/files.crt /etc/ssl/files.key
    root * /srv/files
    file_server browse
}

internal.example.com {
    tls admin@example.com
    reverse_proxy localhost:3000
}`

func caddyByName(t *testing.T, cfg string) (Analysis, map[string]Server) {
	t.Helper()
	a, err := analyzeCaddy(cfg)
	require.NoError(t, err)
	m := map[string]Server{}
	for _, s := range a.Servers {
		for _, n := range s.Names {
			m[n] = s
		}
	}
	return a, m
}

func TestCaddySitesListeners(t *testing.T) {
	a, names := caddyByName(t, caddyCfg)
	require.Len(t, a.Servers, 3) // the global { } block is NOT a server
	require.Contains(t, names, "example.com")
	require.Contains(t, names, "www.example.com")
	require.Contains(t, names, "files.example.com")
	require.Contains(t, names, "internal.example.com")
	require.True(t, names["example.com"].Listeners[0].SSL)
	require.Equal(t, 443, names["example.com"].Listeners[0].Port)
	require.Equal(t, "/var/www/example", names["example.com"].Root)
	require.Equal(t, "/etc/ssl/files.crt", names["files.example.com"].TLSCert)
}

func TestCaddyReverseProxy(t *testing.T) {
	a, names := caddyByName(t, caddyCfg)
	var prox *Location
	ex := names["example.com"]
	for i := range ex.Locations {
		if ex.Locations[i].Kind == "proxy" {
			prox = &ex.Locations[i]
		}
	}
	require.NotNil(t, prox)
	require.Equal(t, "http://10.0.0.5:8080", prox.Target)
	require.Empty(t, with(a, "upstream")) // direct-URL proxy, not flagged
}

func TestCaddyAutoHTTPSNoCertFinding(t *testing.T) {
	a, _ := caddyByName(t, caddyCfg)
	require.Empty(t, with(a, "certificate")) // auto-HTTPS: no missing-cert
	require.NotEmpty(t, with(a, "HSTS"))     // files/internal serve TLS, no HSTS header
}

func TestCaddyBrowseAutoindex(t *testing.T) {
	a, _ := caddyByName(t, caddyCfg)
	require.NotEmpty(t, with(a, "autoindex")) // file_server browse
}

// The block-body reverse_proxy (load-balancer form) must capture a backend.
func TestCaddyReverseProxyBlockBody(t *testing.T) {
	cfg := `example.com {
    reverse_proxy {
        to backend1:80
        to backend2:80
    }
}`
	a, err := analyzeCaddy(cfg)
	require.NoError(t, err)
	require.Len(t, a.Servers, 1)
	var prox *Location
	for i := range a.Servers[0].Locations {
		if a.Servers[0].Locations[i].Kind == "proxy" {
			prox = &a.Servers[0].Locations[i]
		}
	}
	require.NotNil(t, prox)
	require.Equal(t, "backend1:80", prox.Target)
}

// `header -Name` removes a header; it must not be recorded as present.
func TestCaddyHeaderUnsetNotRecorded(t *testing.T) {
	cfg := `secure.example.com {
    tls /c /k
    header -Strict-Transport-Security
}`
	a, err := analyzeCaddy(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, with(a, "HSTS")) // header removed → still missing
}

func TestCaddyUnclosedFinalSite(t *testing.T) {
	cfg := `example.com {
    reverse_proxy localhost:3000`
	a, err := analyzeCaddy(cfg)
	require.NoError(t, err)
	require.Len(t, a.Servers, 1)
}

func TestCaddyNonStandardPortTLS(t *testing.T) {
	cfg := `example.com:8443 {
    reverse_proxy localhost:3000
}`
	a, err := analyzeCaddy(cfg)
	require.NoError(t, err)
	require.True(t, a.Servers[0].Listeners[0].SSL)
}

func TestCaddySecretNotSurfaced(t *testing.T) {
	a, _ := caddyByName(t, caddyCfg)
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "HASHSECRETxyz")
}
