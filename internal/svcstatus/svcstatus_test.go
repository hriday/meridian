package svcstatus

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

const degraded = `{
  "page": {"name": "GitHub"},
  "status": {"indicator": "major", "description": "Partial System Outage"},
  "components": [
    {"name": "Git Operations", "status": "operational"},
    {"name": "Actions", "status": "major_outage"},
    {"name": "Pages", "status": "degraded_performance"}
  ],
  "incidents": [
    {"name": "Actions degraded", "status": "investigating", "impact": "major", "shortlink": "https://stspg.io/x", "updated_at": "2026-11-13T10:00:00Z"},
    {"name": "Old thing", "status": "resolved", "impact": "minor"}
  ]
}`

func TestCheckDistillsOutage(t *testing.T) {
	f := &fakeDoer{body: degraded}
	s, err := Check(context.Background(), f, "github")
	require.NoError(t, err)
	require.Equal(t, "GitHub", s.Service)
	require.Equal(t, "major", s.Indicator)
	require.False(t, s.Operational)
	require.Equal(t, "https://www.githubstatus.com/api/v2/summary.json", f.req.URL.String(), "hits the summary API of the known base")

	// Only non-operational components are surfaced.
	require.Len(t, s.Impacted, 2)
	names := []string{s.Impacted[0].Name, s.Impacted[1].Name}
	require.ElementsMatch(t, []string{"Actions", "Pages"}, names)

	// Resolved incidents are dropped; the active one is kept.
	require.Len(t, s.Incidents, 1)
	require.Equal(t, "Actions degraded", s.Incidents[0].Name)
	require.Equal(t, "https://stspg.io/x", s.Incidents[0].Shortlink)
}

func TestCheckAllOperational(t *testing.T) {
	f := &fakeDoer{body: `{"page":{"name":"Cloudflare"},"status":{"indicator":"none","description":"All Systems Operational"},"components":[{"name":"API","status":"operational"}],"incidents":[]}`}
	s, err := Check(context.Background(), f, "cloudflare")
	require.NoError(t, err)
	require.True(t, s.Operational)
	require.Empty(t, s.Impacted)
	require.Empty(t, s.Incidents)
}

func TestCheckHTTPError(t *testing.T) {
	_, err := Check(context.Background(), &fakeDoer{status: 503}, "github")
	require.Error(t, err)
}

func TestResolveBase(t *testing.T) {
	b, err := resolveBase("github")
	require.NoError(t, err)
	require.Equal(t, "https://www.githubstatus.com", b)

	b, err = resolveBase("https://status.example.com/history")
	require.NoError(t, err)
	require.Equal(t, "https://status.example.com", b, "path is dropped")

	b, err = resolveBase("status.example.com")
	require.NoError(t, err)
	require.Equal(t, "https://status.example.com", b, "bare host gets https")

	_, err = resolveBase("notaservice")
	require.Error(t, err, "unknown single word is an error, not a bad fetch")
}
