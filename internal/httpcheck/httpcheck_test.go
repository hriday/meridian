package httpcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeURL(t *testing.T) {
	got, err := normalizeURL("example.com")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", got)

	got, err = normalizeURL("http://x/y")
	require.NoError(t, err)
	require.Equal(t, "http://x/y", got)

	_, err = normalizeURL("")
	require.Error(t, err)
	_, err = normalizeURL("   ")
	require.Error(t, err)
}

func TestCheckOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "test")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hi"))
	}))
	defer srv.Close()

	r, err := Check(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	require.Equal(t, 200, r.Status)
	require.Contains(t, r.StatusText, "OK")
	require.Equal(t, "test", r.Server)
	require.Equal(t, "text/plain", r.ContentType)
	require.GreaterOrEqual(t, r.Timing.TotalMillis, 0.0)
	require.Empty(t, r.Redirects)
	require.Empty(t, r.TLSVersion) // plain HTTP
}

func TestCheckRedirectChain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/b", 301) })
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	r, err := Check(context.Background(), srv.URL+"/a", 5*time.Second)
	require.NoError(t, err)
	require.Len(t, r.Redirects, 1)
	require.Equal(t, 301, r.Redirects[0].Status)
	require.Equal(t, 200, r.Status)
	require.True(t, strings.HasSuffix(r.FinalURL, "/b"), r.FinalURL)
}

func TestCheckNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	defer srv.Close()
	r, err := Check(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	require.Equal(t, 404, r.Status)
}

func TestCheckRedirectLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/loop", 302) }))
	defer srv.Close()
	_, err := Check(context.Background(), srv.URL+"/loop", 5*time.Second)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "redirect")
}

func TestCheckTLSVersion(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	old := newTransport
	newTransport = func() *http.Transport { return srv.Client().Transport.(*http.Transport).Clone() }
	t.Cleanup(func() { newTransport = old })

	r, err := Check(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	require.Equal(t, 200, r.Status)
	require.True(t, strings.HasPrefix(r.TLSVersion, "TLS"), r.TLSVersion)
	require.GreaterOrEqual(t, r.Timing.TLSMillis, 0.0)
}

func TestCheckDialError(t *testing.T) {
	_, err := Check(context.Background(), "http://127.0.0.1:1", 2*time.Second)
	require.Error(t, err)
}
