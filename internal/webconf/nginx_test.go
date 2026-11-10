package webconf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func extractCfg(t *testing.T, cfg string) Analysis {
	t.Helper()
	dirs, err := parse(cfg)
	require.NoError(t, err)
	return extract(dirs)
}

func TestExtractServersAndListeners(t *testing.T) {
	a := extractCfg(t, `http {
    server {
        server_name example.com www.example.com;
        listen 443 ssl http2;
        root /var/www/example;
    }
    server {
        listen 80 default_server;
        server_name _;
    }
}`)
	require.Len(t, a.Servers, 2)
	require.Equal(t, []string{"example.com", "www.example.com"}, a.Servers[0].Names)
	require.Equal(t, "/var/www/example", a.Servers[0].Root)
	require.Len(t, a.Servers[0].Listeners, 1)
	require.Equal(t, 443, a.Servers[0].Listeners[0].Port)
	require.True(t, a.Servers[0].Listeners[0].SSL)
	require.True(t, a.Servers[0].Listeners[0].HTTP2)
	require.True(t, a.Servers[1].Listeners[0].Default)
	require.Equal(t, 80, a.Servers[1].Listeners[0].Port)
}

func TestExtractLocationsAndProxy(t *testing.T) {
	a := extractCfg(t, `http {
    server {
        location / { try_files $uri =404; }
        location /api/ { proxy_pass http://api; }
        location = /old { return 301 https://example.com/new; }
        location /files/ { alias /srv/files/; }
    }
}`)
	locs := a.Servers[0].Locations
	require.Len(t, locs, 4)
	require.Equal(t, "static", locs[0].Kind)
	require.Equal(t, "proxy", locs[1].Kind)
	require.Equal(t, "api", locs[1].Target) // bare upstream name
	require.Equal(t, "redirect", locs[2].Kind)
	require.Equal(t, "alias", locs[3].Kind)
}

func TestExtractUpstreams(t *testing.T) {
	a := extractCfg(t, `http {
    upstream api {
        least_conn;
        server 10.0.0.1:8080;
        server 10.0.0.2:8080;
    }
}`)
	require.Len(t, a.Upstreams, 1)
	require.Equal(t, "api", a.Upstreams[0].Name)
	require.Equal(t, []string{"10.0.0.1:8080", "10.0.0.2:8080"}, a.Upstreams[0].Servers)
	require.Equal(t, "least_conn", a.Upstreams[0].Policy)
}

func TestExtractTLSAndHeaders(t *testing.T) {
	a := extractCfg(t, `http {
    server_tokens off;
    server {
        listen 443 ssl;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_certificate /etc/ssl/x.crt;
        add_header Strict-Transport-Security "max-age=63072000";
    }
}`)
	s := a.Servers[0]
	require.Equal(t, []string{"TLSv1.2", "TLSv1.3"}, s.TLSProtocols)
	require.Equal(t, "/etc/ssl/x.crt", s.TLSCert)
	require.Contains(t, s.Headers, "Strict-Transport-Security")
}

func TestExtractPlusDirectivesNoError(t *testing.T) {
	a := extractCfg(t, `http {
    keyval_zone zone=one:1m;
    upstream api {
        zone api 64k;
        server 10.0.0.1:8080;
        health_check interval=5s;
        sticky cookie srv;
    }
    server {
        listen 80;
        location /api { api write=on; }
    }
}`)
	require.Len(t, a.Upstreams, 1)
	require.Equal(t, []string{"10.0.0.1:8080"}, a.Upstreams[0].Servers)
	// Plus directives must not themselves be flagged (no errors; nothing names them).
	require.Equal(t, 0, a.Counts.Errors)
	for _, f := range a.Findings {
		for _, plus := range []string{"health_check", "keyval", "sticky", "zone_sync", "api"} {
			require.NotContains(t, f.Title+" "+f.Detail, plus)
		}
	}
}

func TestExtractRealMultiFileDump(t *testing.T) {
	// Real `nginx -T`: nginx.conf has http { include ...; }, and the included
	// vhost/upstream appear as TOP-LEVEL blocks under their own marker.
	a := extractCfg(t, `# configuration file /etc/nginx/nginx.conf:
http {
    server_tokens off;
    include /etc/nginx/sites-enabled/*;
}
# configuration file /etc/nginx/sites-enabled/example.conf:
server {
    listen 443 ssl;
    ssl_certificate /etc/ssl/x.crt;
    server_name example.com;
    location /api { proxy_pass http://api; }
}
upstream api { server 10.0.0.1:8080; }`)
	require.Len(t, a.Servers, 1)
	require.Equal(t, []string{"example.com"}, a.Servers[0].Names)
	require.Len(t, a.Upstreams, 1)
	require.Len(t, a.Upstreams[0].UsedBy, 1) // routing resolved across files
	require.Empty(t, with(a, "server_tokens"))  // off inherited from http
	require.Empty(t, with(a, "undefined"))      // api is defined
}

func TestExtractEmpty(t *testing.T) {
	a := extractCfg(t, ``)
	require.NotNil(t, a.Servers)
	require.Empty(t, a.Servers)
	require.Empty(t, a.Upstreams)
}
