package cli

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/lookup"
	"github.com/stretchr/testify/require"
)

type cliFakeDoer struct{}

func (cliFakeDoer) Do(r *http.Request) (*http.Response, error) {
	body := `{}`
	switch p := r.URL.Path; {
	case strings.Contains(p, "network-info"):
		body = `{"data":{"prefix":"104.16.0.0/13","asns":["13335"]}}`
	case strings.Contains(p, "as-overview"):
		body = `{"data":{"holder":"CLOUDFLARENET, US"}}`
	case strings.Contains(p, "routing-status"):
		body = `{"data":{"announced_space":{"v4":{"prefixes":1542},"v6":{"prefixes":712}},"observed_neighbours":8400}}`
	case strings.Contains(p, "/ip/"):
		body = `{"handle":"NET-X","name":"CLOUDFLARENET","port43":"whois.arin.net","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]}`
	case strings.Contains(p, "/autnum/"):
		body = `{"handle":"AS13335","name":"CLOUDFLARENET","port43":"whois.arin.net","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]}`
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func withFakeLookup(t *testing.T) {
	old := newLookupClient
	newLookupClient = func() *lookup.Client { return lookup.NewClient(cliFakeDoer{}, time.Minute) }
	t.Cleanup(func() { newLookupClient = old })
}

func TestRunLookupJSON(t *testing.T) {
	withFakeLookup(t)
	var out, errb bytes.Buffer
	code := Run([]string{"lookup", "104.16.123.45", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), `"originAsn":"AS13335"`)
	require.Contains(t, out.String(), "Cloudflare, Inc.")
}

func TestRunLookupRoutesASN(t *testing.T) {
	withFakeLookup(t)
	var out, errb bytes.Buffer
	code := Run([]string{"lookup", "AS13335", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), `"v4Prefixes":1542`)
	require.Contains(t, out.String(), "CLOUDFLARENET")
}

func TestRunLookupRejectsNonIPASN(t *testing.T) {
	withFakeLookup(t)
	var out, errb bytes.Buffer
	code := Run([]string{"lookup", "example.com"}, &out, &errb)
	require.NotEqual(t, 0, code)
	require.NotEmpty(t, errb.String())
}
