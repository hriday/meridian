package ctlog

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMarkStale(t *testing.T) {
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	subs := []Subdomain{
		{Name: "old.example.com", LastSeen: "2017-04-07"},    // ~9y ago → stale
		{Name: "recent.example.com", LastSeen: "2026-06-01"}, // this year → active
		{Name: "edge.example.com", LastSeen: "2024-11-30"},   // ~1y ago, under 2y → active
		{Name: "nodate.example.com", LastSeen: ""},           // unparseable → left alone
	}
	markStale(subs, now)
	require.True(t, subs[0].Stale, "a cert that expired 9 years ago is stale")
	require.False(t, subs[1].Stale, "a cert from this year is active")
	require.False(t, subs[2].Stale, "just under the 2-year threshold stays active")
	require.False(t, subs[3].Stale, "an unparseable date is never flagged")
}

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
	// Both sources fail (the fake returns 502 for every host) → error.
	f := &fakeDoer{status: 502, body: ""}
	_, err := Subdomains(context.Background(), f, "example.com")
	require.Error(t, err)
}

type routeResp struct {
	status int
	body   string
}

type routeDoer struct{ m map[string]routeResp }

func (d *routeDoer) Do(req *http.Request) (*http.Response, error) {
	r := d.m[req.URL.Host]
	st := r.status
	if st == 0 {
		st = 500
	}
	return &http.Response{StatusCode: st, Body: io.NopCloser(strings.NewReader(r.body)), Header: make(http.Header)}, nil
}

func TestSubdomainsFallsBackToCertSpotter(t *testing.T) {
	// crt.sh errors (502); certSpotter answers → its names are used.
	d := &routeDoer{m: map[string]routeResp{
		"crt.sh": {status: 502},
		"api.certspotter.com": {status: 200, body: `[
		  {"dns_names":["www.example.com","api.example.com"],"not_before":"2024-01-01T00:00:00Z","not_after":"2024-04-01T00:00:00Z"},
		  {"dns_names":["other.notours.com"],"not_before":"2024-01-01T00:00:00Z","not_after":"2024-02-01T00:00:00Z"}
		]`},
	}}
	subs, err := Subdomains(context.Background(), d, "example.com")
	require.NoError(t, err)
	names := map[string]bool{}
	for _, s := range subs {
		names[s.Name] = true
	}
	require.True(t, names["www.example.com"], "should include www from certSpotter")
	require.True(t, names["api.example.com"], "should include api from certSpotter")
	require.False(t, names["other.notours.com"], "out-of-scope name dropped")
}
