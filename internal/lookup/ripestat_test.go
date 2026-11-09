package lookup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ripeServer returns an httptest server that serves canned RIPEstat payloads
// chosen by the request path, and points ripeBase at it for the test.
func ripeServer(t *testing.T, payloads map[string]string) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for frag, body := range payloads {
			if strings.Contains(r.URL.Path, frag) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
				return
			}
		}
		http.Error(w, "no fixture", 404)
	}))
	t.Cleanup(srv.Close)
	old := ripeBase
	ripeBase = srv.URL + "/"
	t.Cleanup(func() { ripeBase = old })
	return NewClient(http.DefaultClient, time.Minute)
}

func TestRipeNetworkInfoParsesPrefixAndOrigin(t *testing.T) {
	c := ripeServer(t, map[string]string{
		"network-info": `{"data":{"prefix":"104.16.0.0/13","asns":["13335"]}}`,
	})
	prefix, origin, err := c.ripeNetworkInfo(context.Background(), "104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "104.16.0.0/13", prefix)
	require.Equal(t, []string{"13335"}, origin)
}

func TestRipeRoutingStatusCounts(t *testing.T) {
	c := ripeServer(t, map[string]string{
		"routing-status": `{"data":{"announced_space":{"v4":{"prefixes":1542},"v6":{"prefixes":712}},"observed_neighbours":8400}}`,
	})
	v4, v6, nb, err := c.ripeRoutingStatus(context.Background(), "AS13335")
	require.NoError(t, err)
	require.Equal(t, 1542, v4)
	require.Equal(t, 712, v6)
	require.Equal(t, 8400, nb)
}

func TestRipeASOverviewHolder(t *testing.T) {
	c := ripeServer(t, map[string]string{
		"as-overview": `{"data":{"holder":"CLOUDFLARENET, US"}}`,
	})
	holder, err := c.ripeASOverview(context.Background(), "AS13335")
	require.NoError(t, err)
	require.Equal(t, "CLOUDFLARENET, US", holder)
}

func TestRipeNetworkInfoNoOrigin(t *testing.T) {
	c := ripeServer(t, map[string]string{
		"network-info": `{"data":{"prefix":"","asns":[]}}`,
	})
	prefix, origin, err := c.ripeNetworkInfo(context.Background(), "192.0.2.1")
	require.NoError(t, err)
	require.Equal(t, "", prefix)
	require.Empty(t, origin)
}
