package cve

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

const log4shell = `{"vulnerabilities":[{"cve":{
  "id":"CVE-2021-44228","published":"2021-12-10T10:15:09.143","cisaExploitAdd":"2021-12-10",
  "descriptions":[{"lang":"es","value":"ignore"},{"lang":"en","value":"Apache Log4j2 JNDI RCE."}],
  "metrics":{"cvssMetricV31":[{"cvssData":{"baseScore":10.0,"baseSeverity":"CRITICAL","vectorString":"CVSS:3.1/AV:N/AC:L/PR:N"}}]},
  "weaknesses":[{"description":[{"value":"CWE-917"},{"value":"CWE-917"},{"value":"CWE-20"}]}],
  "references":[{"url":"https://logging.apache.org/"},{"url":"https://logging.apache.org/"},{"url":"https://nvd.example/x"}]
}}]}`

func TestLookupFlattens(t *testing.T) {
	f := &fakeDoer{body: log4shell}
	c, err := Lookup(context.Background(), f, "cve-2021-44228")
	require.NoError(t, err)
	require.Equal(t, "CVE-2021-44228", c.ID)
	require.Equal(t, "2021-12-10", c.Published)
	require.Equal(t, "Apache Log4j2 JNDI RCE.", c.Description, "English description chosen")
	require.Equal(t, 10.0, c.CVSS)
	require.Equal(t, "CRITICAL", c.Severity)
	require.True(t, c.KEV, "flagged as CISA known-exploited")
	require.ElementsMatch(t, []string{"CWE-917", "CWE-20"}, c.CWE, "CWEs deduped")
	require.Len(t, c.References, 2, "references deduped")
	require.Equal(t, "CVE-2021-44228", f.req.URL.Query().Get("cveId"), "queried by cveId")
}

func TestLookupRejectsNonID(t *testing.T) {
	f := &fakeDoer{}
	_, err := Lookup(context.Background(), f, "log4j")
	require.Error(t, err)
	require.Nil(t, f.req, "no request for a non-id")
}

func TestLookupNotFound(t *testing.T) {
	f := &fakeDoer{body: `{"vulnerabilities":[]}`}
	_, err := Lookup(context.Background(), f, "CVE-1999-0001")
	require.Error(t, err)
}

func TestSearchByKeyword(t *testing.T) {
	f := &fakeDoer{body: log4shell}
	cves, err := Search(context.Background(), f, "log4j")
	require.NoError(t, err)
	require.Len(t, cves, 1)
	require.Equal(t, "log4j", f.req.URL.Query().Get("keywordSearch"))
}

func TestSearchEmpty(t *testing.T) {
	_, err := Search(context.Background(), &fakeDoer{}, "   ")
	require.Error(t, err)
}

func TestRateLimited(t *testing.T) {
	f := &fakeDoer{status: 403, body: ""}
	_, err := Search(context.Background(), f, "log4j")
	require.Error(t, err)
	require.Contains(t, err.Error(), "rate-limit")
}

func TestIsID(t *testing.T) {
	require.True(t, IsID("CVE-2021-44228"))
	require.True(t, IsID("cve-1999-0001"))
	require.False(t, IsID("log4j"))
	require.False(t, IsID("CVE-21-1"))
}
