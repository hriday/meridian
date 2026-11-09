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

type prefixDoer struct{}

func (prefixDoer) Do(r *http.Request) (*http.Response, error) {
	body := `{}`
	switch p := r.URL.Path; {
	case strings.Contains(p, "prefix-overview"):
		body = `{"data":{"announced":true,"asns":[{"asn":13335,"holder":"CLOUDFLARENET, US"}]}}`
	case strings.Contains(p, "related-prefixes"):
		body = `{"data":{"prefixes":[]}}`
	case strings.Contains(p, "rpki-validation"):
		body = `{"data":{"status":"valid","validating_roas":[]}}`
	case strings.Contains(p, "/ip/"):
		body = `{"handle":"NET-X","name":"CLOUDFLARENET","port43":"whois.arin.net","entities":[{"roles":["registrant"],"vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]}`
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func TestRunLookupPrefixJSON(t *testing.T) {
	old := newLookupClient
	t.Cleanup(func() { newLookupClient = old })
	newLookupClient = func() *lookup.Client { return lookup.NewClient(prefixDoer{}, time.Minute) }

	var out, errb bytes.Buffer
	code := Run([]string{"lookup", "104.16.0.0/13", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), `"originAsn":"AS13335"`)
	require.Contains(t, out.String(), `"rpkiState":"valid"`)
}
