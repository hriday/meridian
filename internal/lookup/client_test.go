package lookup_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/lookup"
	"github.com/stretchr/testify/require"
)

// doerFunc adapts a function to the lookup.Doer interface.
type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestGetJSONCachesAndSetsUserAgent(t *testing.T) {
	var got *http.Request
	d := doerFunc(func(r *http.Request) (*http.Response, error) {
		if got != nil {
			t.Fatal("second network call for a cached url")
		}
		got = r
		return jsonResp(200, `{"n":7}`), nil
	})
	c := lookup.NewClient(d, time.Minute)
	var out struct{ N int }
	require.NoError(t, c.GetJSON(context.Background(), "https://x/y", &out))
	require.Equal(t, 7, out.N)
	require.Contains(t, got.Header.Get("User-Agent"), "meridian/")
	// Second call for the same url must be served from cache (doer would t.Fatal).
	out.N = 0
	require.NoError(t, c.GetJSON(context.Background(), "https://x/y", &out))
	require.Equal(t, 7, out.N)
}

func TestGetJSONErrorsOnNon200(t *testing.T) {
	d := doerFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResp(500, `{"err":"boom"}`), nil
	})
	c := lookup.NewClient(d, time.Minute)
	var out struct{ N int }
	err := c.GetJSON(context.Background(), "https://x/fail", &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestGetJSONErrorsOnBadBody(t *testing.T) {
	d := doerFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResp(200, `{not valid json`), nil
	})
	c := lookup.NewClient(d, time.Minute)
	var out struct{ N int }
	err := c.GetJSON(context.Background(), "https://x/bad", &out)
	require.Error(t, err)
}

func TestPeeringNotListedOn404(t *testing.T) {
	d := doerFunc(func(*http.Request) (*http.Response, error) {
		return jsonResp(404, `{"message":"Not Found"}`), nil
	})
	p, err := lookup.NewClient(d, time.Minute).Peering(context.Background(), "399098")
	require.NoError(t, err, "a 404 means the ASN isn't listed, not an outage")
	require.Nil(t, p)
}

func TestPeeringErrorsOnOutage(t *testing.T) {
	d := doerFunc(func(*http.Request) (*http.Response, error) {
		return jsonResp(503, `unavailable`), nil
	})
	_, err := lookup.NewClient(d, time.Minute).Peering(context.Background(), "13335")
	require.Error(t, err, "a real outage (5xx) is still surfaced as an error")
}
