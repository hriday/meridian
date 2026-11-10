package webconf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNestedBlocks(t *testing.T) {
	dirs, err := parse(`http {
    server {
        listen 80;
        location / {
            return 200;
        }
    }
}`)
	require.NoError(t, err)
	require.Len(t, dirs, 1)
	http := dirs[0]
	require.Equal(t, "http", http.Name)
	srv := http.children("server")
	require.Len(t, srv, 1)
	require.Len(t, srv[0].children("listen"), 1)
	require.Equal(t, "80", srv[0].children("listen")[0].arg(0))
	loc := srv[0].children("location")
	require.Len(t, loc, 1)
	require.Equal(t, "/", loc[0].arg(0))
	require.Len(t, loc[0].children("return"), 1)
}

func TestParseDirectiveArgsAndQuotes(t *testing.T) {
	dirs, err := parse(`add_header X-Frame-Options "SAMEORIGIN";
ssl_protocols TLSv1.2 TLSv1.3;`)
	require.NoError(t, err)
	require.Equal(t, "add_header", dirs[0].Name)
	require.Equal(t, "X-Frame-Options", dirs[0].arg(0))
	require.Equal(t, "SAMEORIGIN", dirs[0].arg(1))
	require.Equal(t, []string{"TLSv1.2", "TLSv1.3"}, dirs[1].Args)
}

func TestParseFileMarkersAndPreamble(t *testing.T) {
	dirs, err := parse(`nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful
# configuration file /etc/nginx/nginx.conf:
worker_processes auto;
# configuration file /etc/nginx/sites-enabled/site.conf:
server {
    listen 80;
}`)
	require.NoError(t, err)
	require.Len(t, dirs, 2) // worker_processes + server; preamble ignored
	require.Equal(t, "worker_processes", dirs[0].Name)
	require.Equal(t, "/etc/nginx/nginx.conf", dirs[0].File)
	require.Equal(t, "server", dirs[1].Name)
	require.Equal(t, "/etc/nginx/sites-enabled/site.conf", dirs[1].File)
	require.Equal(t, 2, dirs[1].Line) // line within its file (marker is line 1)
}

func TestParseUnclosedBlockErrors(t *testing.T) {
	_, err := parse(`http {
    server {
        listen 80;
}`)
	require.Error(t, err)
}
