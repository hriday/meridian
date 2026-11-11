package webconf

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic Apache httpd config: global ServerTokens, an http→https redirect
// vhost, and a TLS vhost with names, root, reverse proxy, hardened SSLProtocol,
// cert, security headers, an autoindex Directory, and a bind-password the parser
// must ignore.
const apacheCfg = `ServerTokens Prod
ServerSignature Off
Listen 80
Listen 443

<VirtualHost *:80>
    ServerName example.com
    ServerAlias www.example.com
    Redirect permanent / https://example.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName example.com
    ServerAlias www.example.com
    DocumentRoot /var/www/example
    SSLEngine on
    SSLProtocol all -SSLv3 -TLSv1 -TLSv1.1
    SSLCertificateFile /etc/ssl/certs/example.crt
    SSLCertificateKeyFile /etc/ssl/private/example.key
    AuthLDAPBindPassword SUPERSECRET123
    Header always set Strict-Transport-Security "max-age=63072000"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "SAMEORIGIN"
    ProxyPass /api http://10.0.0.5:8080/
    ProxyPassReverse /api http://10.0.0.5:8080/
    <Directory /var/www/example>
        Options Indexes FollowSymLinks
    </Directory>
</VirtualHost>`

func apacheServers(t *testing.T, cfg string) (Analysis, *Server, *Server) {
	t.Helper()
	a, err := analyzeApache(cfg)
	require.NoError(t, err)
	var s80, s443 *Server
	for i := range a.Servers {
		for _, l := range a.Servers[i].Listeners {
			if l.Port == 80 {
				s80 = &a.Servers[i]
			}
			if l.Port == 443 {
				s443 = &a.Servers[i]
			}
		}
	}
	return a, s80, s443
}

func TestApacheVHostsListeners(t *testing.T) {
	a, s80, s443 := apacheServers(t, apacheCfg)
	require.Len(t, a.Servers, 2)
	require.NotNil(t, s80)
	require.NotNil(t, s443)
	require.Contains(t, s443.Names, "example.com")
	require.Contains(t, s443.Names, "www.example.com")
	require.Equal(t, "/var/www/example", s443.Root)
	require.Equal(t, "/etc/ssl/certs/example.crt", s443.TLSCert)
	require.True(t, s443.Listeners[0].SSL)
	require.False(t, s80.Listeners[0].SSL)
}

func TestApacheProxyLocation(t *testing.T) {
	a, _, s443 := apacheServers(t, apacheCfg)
	var prox *Location
	for i := range s443.Locations {
		if s443.Locations[i].Kind == "proxy" {
			prox = &s443.Locations[i]
		}
	}
	require.NotNil(t, prox)
	require.Equal(t, "/api", prox.Path)
	require.Equal(t, "http://10.0.0.5:8080/", prox.Target)
	require.Empty(t, with(a, "upstream")) // direct-URL proxy, not flagged
}

func TestApacheTLSAndHeaders(t *testing.T) {
	a, _, _ := apacheServers(t, apacheCfg)
	require.Empty(t, with(a, "HSTS"))           // header present
	require.Empty(t, with(a, "X-Content-Type")) // header present
	require.Empty(t, with(a, "Apache version")) // ServerTokens Prod
	require.Empty(t, with(a, "certificate"))    // cert set
	require.Empty(t, with(a, "TLSv1"))          // TLSv1/1.1 disabled via -TLSv1
	require.NotEmpty(t, with(a, "autoindex"))   // Directory Options Indexes
}

func TestApacheDeprecatedTLS(t *testing.T) {
	cfg := `<VirtualHost *:443>
    ServerName x
    SSLEngine on
    SSLProtocol TLSv1.2 TLSv1
    SSLCertificateFile /c
</VirtualHost>`
	a, err := analyzeApache(cfg)
	require.NoError(t, err)
	require.NotEmpty(t, with(a, "TLSv1")) // TLSv1 explicitly enabled
}

func TestApacheSecretNotSurfaced(t *testing.T) {
	a, _, _ := apacheServers(t, apacheCfg)
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "SUPERSECRET123")
}
