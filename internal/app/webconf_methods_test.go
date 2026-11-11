package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const nginxSample = `# configuration file /etc/nginx/nginx.conf:
http {
    upstream api { server 10.0.0.1:8080; }
    server {
        listen 443 ssl;
        ssl_certificate /etc/ssl/x.crt;
        server_name example.com;
        location /api { proxy_pass http://api; }
        location /bad { proxy_pass http://nope; }
    }
}`

func TestAnalyzeWebServerEmpty(t *testing.T) {
	_, err := New().AnalyzeWebServer("", "nginx")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "config")
}

func TestAnalyzeWebServerNginx(t *testing.T) {
	a, err := New().AnalyzeWebServer(nginxSample, "nginx")
	require.NoError(t, err)
	require.Len(t, a.Servers, 1)
	require.Equal(t, []string{"example.com"}, a.Servers[0].Names)
	require.GreaterOrEqual(t, a.Counts.Errors, 1) // undefined upstream "nope"
}

func TestAnalyzeWebServerUnknownServer(t *testing.T) {
	_, err := New().AnalyzeWebServer("http {}", "lighttpd")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "support")
}

func TestAnalyzeWebServerDefault(t *testing.T) {
	a, err := New().AnalyzeWebServer(nginxSample, "")
	require.NoError(t, err)
	require.Equal(t, "nginx", a.ServerType)
}

func TestAnalyzeUnresolvedInclude(t *testing.T) {
	cfg := `http {
    include /etc/nginx/sites-enabled/*.conf;
    server { listen 80; server_name x; }
}`
	a, err := New().AnalyzeWebServer(cfg, "nginx")
	require.NoError(t, err)
	found := false
	for _, f := range a.Findings {
		if strings.Contains(f.Title+" "+f.Detail, "include") {
			found = true
		}
	}
	require.True(t, found, "should flag the unresolved include")
}
