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

type bgpDoer struct{}

func (bgpDoer) Do(r *http.Request) (*http.Response, error) {
	body := `{}`
	switch p := r.URL.Path; {
	case strings.Contains(p, "looking-glass"):
		body = `{"data":{"rrcs":[{"rrc":"RRC01","location":"London","scope":"LINX","peers":[
			{"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"}]}]}}`
	case strings.Contains(p, "rpki-validation"):
		body = `{"data":{"status":"valid","validating_roas":[]}}`
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func TestAppLookupBGP(t *testing.T) {
	a := New()
	a.lookup = lookup.NewClient(bgpDoer{}, time.Minute)
	r, err := a.LookupBGP("140.78.0.0/16")
	require.NoError(t, err)
	require.Greater(t, r.Collectors, 0)
	require.Equal(t, []string{"1205"}, r.OriginASNs)
	require.Equal(t, "valid", r.RPKIState)
}
