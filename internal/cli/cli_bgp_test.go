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

type bgpDoer struct{}

func (bgpDoer) Do(r *http.Request) (*http.Response, error) {
	body := `{}`
	switch p := r.URL.Path; {
	case strings.Contains(p, "looking-glass"):
		body = `{"data":{"rrcs":[{"rrc":"RRC01","location":"London","scope":"LINX","peers":[
			{"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"},
			{"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"}]}]}}`
	case strings.Contains(p, "rpki-validation"):
		body = `{"data":{"status":"valid","validating_roas":[]}}`
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func withBGPClient(t *testing.T) {
	old := newLookupClient
	t.Cleanup(func() { newLookupClient = old })
	newLookupClient = func() *lookup.Client { return lookup.NewClient(bgpDoer{}, time.Minute) }
}

func TestRunBGPPrintsSummary(t *testing.T) {
	withBGPClient(t)
	var out, errb bytes.Buffer
	code := Run([]string{"bgp", "140.78.0.0/16"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	s := out.String()
	require.Contains(t, s, "140.78.0.0/16")
	require.Contains(t, s, "AS1205")          // origin
	require.Contains(t, s, "seen by 2 peers") // visibility
	require.Contains(t, s, "AS174 → AS1205")  // the top path
	require.Contains(t, s, "valid")           // RPKI
}

func TestRunBGPJSON(t *testing.T) {
	withBGPClient(t)
	var out, errb bytes.Buffer
	code := Run([]string{"bgp", "140.78.0.0/16", "--json"}, &out, &errb)
	require.Equal(t, 0, code, errb.String())
	require.Contains(t, out.String(), `"originAsns":["1205"]`)
}

func TestRunBGPRejectsNonPrefix(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"bgp", "not-an-address"}, &out, &errb)
	require.Equal(t, 2, code)
}
