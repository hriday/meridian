package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPCheckMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	r, err := New().HTTPCheck(srv.URL)
	require.NoError(t, err)
	require.Equal(t, 200, r.Status)
	require.Equal(t, srv.URL, r.FinalURL)
}
