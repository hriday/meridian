package app

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/lookup"
	"github.com/stretchr/testify/require"
)

// appFakeDoer serves canned RDAP/RIPEstat bodies by request path, so the
// bound methods can be tested without the network.
type appFakeDoer struct{}

func (appFakeDoer) Do(r *http.Request) (*http.Response, error) {
	body := `{}`
	switch p := r.URL.Path; {
	case strings.Contains(p, "network-info"):
		body = `{"data":{"prefix":"104.16.0.0/13","asns":["13335"]}}`
	case strings.Contains(p, "as-overview"):
		body = `{"data":{"holder":"CLOUDFLARENET, US"}}`
	case strings.Contains(p, "/ip/"):
		body = `{"handle":"NET-X","name":"CLOUDFLARENET","port43":"whois.arin.net","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]}`
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func TestAppLookupIP(t *testing.T) {
	a := New()
	a.lookup = lookup.NewClient(appFakeDoer{}, time.Minute)
	r, err := a.LookupIP("104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "AS13335", r.OriginASN)
	require.Equal(t, "Cloudflare, Inc.", r.RegisteredOrg)
}
