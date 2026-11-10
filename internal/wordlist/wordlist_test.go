package wordlist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func sha256hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func serve(t *testing.T, body []byte) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadVerifies(t *testing.T) {
	body := []byte("password\n123456\nqwerty\n")
	srv := serve(t, body)
	dir := t.TempDir()
	e := Entry{Name: "test.txt", URL: srv.URL, SHA256: sha256hex(body)}

	done := int64(0)
	path, err := Download(context.Background(), e, dir, srv.Client(), func(d, _ int64) { done = d })
	require.NoError(t, err)
	require.FileExists(t, path)
	got, _ := os.ReadFile(path)
	require.Equal(t, body, got)
	require.Equal(t, int64(len(body)), done)
}

func TestDownloadChecksumMismatch(t *testing.T) {
	srv := serve(t, []byte("tampered content"))
	dir := t.TempDir()
	e := Entry{Name: "bad.txt", URL: srv.URL,
		SHA256: "0000000000000000000000000000000000000000000000000000000000000000"}

	_, err := Download(context.Background(), e, dir, srv.Client(), nil)
	require.Error(t, err)
	// the partial file must not be left behind
	entries, _ := os.ReadDir(dir)
	require.Empty(t, entries)
}

func TestDownloadRejectsEmptyChecksum(t *testing.T) {
	srv := serve(t, []byte("whatever"))
	dir := t.TempDir()
	e := Entry{Name: "noverify.txt", URL: srv.URL} // no SHA256
	_, err := Download(context.Background(), e, dir, srv.Client(), nil)
	require.Error(t, err)
	entries, _ := os.ReadDir(dir)
	require.Empty(t, entries) // nothing kept unverified
}

func TestInstalled(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rockyou.txt"), []byte("x"), 0o600))
	got, err := Installed(dir)
	require.NoError(t, err)
	require.Contains(t, got, "rockyou.txt")
}

func TestCatalogHasChecksums(t *testing.T) {
	cat := Catalog()
	require.NotEmpty(t, cat)
	for _, e := range cat {
		require.NotEmpty(t, e.Name, "entry name")
		require.NotEmpty(t, e.URL, "%s URL", e.Name)
		require.Len(t, e.SHA256, 64, "%s needs a pinned sha256", e.Name)
	}
}
