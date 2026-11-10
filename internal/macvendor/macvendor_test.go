package macvendor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSpellings(t *testing.T) {
	for _, in := range []string{
		"fc:fb:fb",
		"fc-fb-fb",
		"fcfb.fbdd.eeff",
		"fcfbfbddeeff",
		"FC:FB:FB:DD:EE:FF",
	} {
		oui, query, _, err := normalize(in)
		require.NoError(t, err, in)
		require.Equal(t, "FC:FB:FB", oui, in)
		require.Equal(t, "FCFBFB", query, in)
	}
}

func TestNormalizeRejects(t *testing.T) {
	for _, in := range []string{"", "fc:fb", "fcfb", "zz:fb:fb", "   "} {
		_, _, _, err := normalize(in)
		require.Errorf(t, err, "expected error for %q", in)
	}
}

func TestNormalizeLocalAdmin(t *testing.T) {
	for _, in := range []string{"02:00:00", "06:aa:bb", "0a:aa:bb", "0e:aa:bb"} {
		_, _, local, err := normalize(in)
		require.NoError(t, err, in)
		require.True(t, local, in)
	}
	for _, in := range []string{"fc:fb:fb", "00:11:22", "08:aa:bb"} {
		_, _, local, err := normalize(in)
		require.NoError(t, err, in)
		require.False(t, local, in)
	}
}

func serve(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	old := baseURL
	baseURL = srv.URL + "/"
	t.Cleanup(func() { baseURL = old })
	return srv
}

func TestLookupFound(t *testing.T) {
	srv := serve(t, 200, "Apple, Inc.")
	r, err := Lookup(context.Background(), srv.Client(), "fc:fb:fb:dd:ee:ff")
	require.NoError(t, err)
	require.True(t, r.Found)
	require.Equal(t, "Apple, Inc.", r.Vendor)
	require.Equal(t, "FC:FB:FB", r.OUI)
}

func TestLookupNotFound(t *testing.T) {
	srv := serve(t, 404, "Not Found")
	r, err := Lookup(context.Background(), srv.Client(), "02:00:00:00:00:00")
	require.NoError(t, err)
	require.False(t, r.Found)
	require.True(t, r.LocallyAdministered)
}

func TestLookupRateLimited(t *testing.T) {
	srv := serve(t, 429, "Too Many Requests")
	_, err := Lookup(context.Background(), srv.Client(), "fc:fb:fb")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "second")
}

type errDoer struct{}

func (errDoer) Do(*http.Request) (*http.Response, error) { return nil, errors.New("boom") }

func TestLookupNetworkError(t *testing.T) {
	_, err := Lookup(context.Background(), errDoer{}, "fc:fb:fb")
	require.Error(t, err)
}

func TestLookupRejectsBadInput(t *testing.T) {
	_, err := Lookup(context.Background(), errDoer{}, "zz")
	require.Error(t, err) // normalized before any request
}
