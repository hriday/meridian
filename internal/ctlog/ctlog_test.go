package ctlog

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeDoer struct {
	body   string
	status int
	req    *http.Request
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.req = req
	st := f.status
	if st == 0 {
		st = 200
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader(f.body)), Header: make(http.Header)}, nil
}

func TestSubdomainsAggregates(t *testing.T) {
	body := `[
	  {"common_name":"api.example.com","name_value":"api.example.com\nwww.example.com","not_before":"2023-01-01T00:00:00","not_after":"2023-04-01T00:00:00"},
	  {"common_name":"api.example.com","name_value":"api.example.com","not_before":"2024-02-01T00:00:00","not_after":"2024-05-01T00:00:00"},
	  {"common_name":"*.cdn.example.com","name_value":"*.cdn.example.com","not_before":"2022-06-01T00:00:00","not_after":"2022-09-01T00:00:00"},
	  {"common_name":"other.notours.com","name_value":"other.notours.com","not_before":"2024-01-01T00:00:00","not_after":"2024-02-01T00:00:00"},
	  {"common_name":"example.com","name_value":"example.com","not_before":"2021-01-01T00:00:00","not_after":"2021-02-01T00:00:00"}
	]`
	f := &fakeDoer{body: body}
	subs, err := Subdomains(context.Background(), f, "example.com")
	require.NoError(t, err)

	names := map[string]Subdomain{}
	for _, s := range subs {
		names[s.Name] = s
	}
	require.Contains(t, names, "example.com")
	require.Contains(t, names, "api.example.com")
	require.Contains(t, names, "www.example.com")
	require.Contains(t, names, "cdn.example.com")
	require.NotContains(t, names, "other.notours.com", "out-of-scope names are excluded")

	// api seen in 2 certs, first/last span both; wildcard flagged on cdn.
	require.Equal(t, 2, names["api.example.com"].Certs)
	require.Equal(t, "2023-01-01", names["api.example.com"].FirstSeen)
	require.Equal(t, "2024-05-01", names["api.example.com"].LastSeen)
	require.True(t, names["cdn.example.com"].Wildcard)
	require.False(t, names["api.example.com"].Wildcard)

	// sorted by name
	require.True(t, subs[0].Name <= subs[len(subs)-1].Name)
	// crt.sh query used the wildcard + json output
	require.Equal(t, "%.example.com", f.req.URL.Query().Get("q"))
	require.Equal(t, "json", f.req.URL.Query().Get("output"))
	require.NotEmpty(t, f.req.Header.Get("User-Agent"))
}

func TestSubdomainsValidation(t *testing.T) {
	f := &fakeDoer{}
	for _, bad := range []string{"", "   ", "notadomain", "has space.com"} {
		_, err := Subdomains(context.Background(), f, bad)
		require.Error(t, err, bad)
	}
	require.Nil(t, f.req, "no request for an invalid domain")
}

func TestSubdomainsNormalizesInput(t *testing.T) {
	f := &fakeDoer{body: "[]"}
	_, err := Subdomains(context.Background(), f, "https://*.Example.com/path")
	require.NoError(t, err)
	require.Equal(t, "%.example.com", f.req.URL.Query().Get("q"))
}

func TestSubdomainsHTTPError(t *testing.T) {
	f := &fakeDoer{status: 502, body: ""}
	_, err := Subdomains(context.Background(), f, "example.com")
	require.Error(t, err)
}
