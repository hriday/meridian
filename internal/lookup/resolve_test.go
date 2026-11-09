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

type srvResp struct {
	code int
	body string
}

// lookupServer serves fixtures chosen by path fragment and points both the
// RIPEstat and RDAP bases at it.
func lookupServer(t *testing.T, fixtures map[string]srvResp) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for frag, f := range fixtures {
			if strings.Contains(r.URL.Path, frag) {
				if f.code != 0 {
					w.WriteHeader(f.code)
				}
				_, _ = w.Write([]byte(f.body))
				return
			}
		}
		http.Error(w, "no fixture: "+r.URL.Path, 404)
	}))
	t.Cleanup(srv.Close)
	or, od := ripeBase, rdapBase
	ripeBase, rdapBase = srv.URL+"/", srv.URL+"/"
	t.Cleanup(func() { ripeBase, rdapBase = or, od })
	return NewClient(http.DefaultClient, time.Minute)
}

type failDoer struct{ t *testing.T }

func (f failDoer) Do(*http.Request) (*http.Response, error) {
	f.t.Fatal("unexpected network call for a non-global address")
	return nil, nil
}

func TestLookupIPCombinesSources(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"network-info": {body: `{"data":{"prefix":"104.16.0.0/13","asns":["13335"]}}`},
		"as-overview":  {body: `{"data":{"holder":"CLOUDFLARENET, US"}}`},
		"/ip/":         {body: arinIPBody},
	})
	r, err := c.LookupIP(context.Background(), "104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "IPv4", r.Version)
	require.Equal(t, "104.16.0.0/13", r.CoveringPrefix)
	require.Equal(t, "AS13335", r.OriginASN)
	require.Equal(t, "CLOUDFLARENET, US", r.OriginName)
	require.Equal(t, "Cloudflare, Inc.", r.RegisteredOrg)
	require.Equal(t, "abuse@cloudflare.com", r.AbuseEmail)
	require.Equal(t, "ARIN", r.Registry)
	require.Empty(t, r.RoutingSrc.Err)
	require.Empty(t, r.RegistrySrc.Err)
	require.False(t, r.RoutingSrc.FetchedAt.IsZero())
}

func TestLookupIPPrivateAddress(t *testing.T) {
	c := NewClient(failDoer{t}, time.Minute) // any network call fails the test
	r, err := c.LookupIP(context.Background(), "10.1.2.3")
	require.NoError(t, err)
	require.Equal(t, "IPv4", r.Version)
	require.Empty(t, r.OriginASN)
	require.NotEmpty(t, r.RoutingNote)
}

func TestLookupPartialOnSourceError(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"network-info": {code: 500, body: `{}`},
		"/ip/":         {body: arinIPBody},
	})
	r, err := c.LookupIP(context.Background(), "104.16.123.45")
	require.NoError(t, err)                      // partial, not fatal
	require.NotEmpty(t, r.RoutingSrc.Err)        // routing failed
	require.Empty(t, r.OriginASN)                // no routing data
	require.Equal(t, "Cloudflare, Inc.", r.RegisteredOrg) // registry still populated
	require.Empty(t, r.RegistrySrc.Err)
}

func TestLookupIPv6(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"network-info": {body: `{"data":{"prefix":"2606:4700::/32","asns":["13335"]}}`},
		"as-overview":  {body: `{"data":{"holder":"CLOUDFLARENET, US"}}`},
		"/ip/":         {body: arinIPBody},
	})
	r, err := c.LookupIP(context.Background(), "2606:4700::6810:84e5")
	require.NoError(t, err)
	require.Equal(t, "IPv6", r.Version)
	require.Equal(t, "2606:4700::/32", r.CoveringPrefix)
}

func TestLookupInvalidIP(t *testing.T) {
	c := NewClient(failDoer{t}, time.Minute)
	_, err := c.LookupIP(context.Background(), "not-an-ip")
	require.Error(t, err)
}

func TestLookupASNCombines(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"routing-status": {body: `{"data":{"announced_space":{"v4":{"prefixes":1542},"v6":{"prefixes":712}},"observed_neighbours":8400}}`},
		"as-overview":    {body: `{"data":{"holder":"CLOUDFLARENET, US"}}`},
		"/autnum/":       {body: arinASNBody},
	})
	r, err := c.LookupASN(context.Background(), "AS13335")
	require.NoError(t, err)
	require.Equal(t, "AS13335", r.ASN)
	require.Equal(t, "CLOUDFLARENET, US", r.Holder)
	require.Equal(t, 1542, r.V4Prefixes)
	require.Equal(t, 712, r.V6Prefixes)
	require.Equal(t, 8400, r.Neighbours)
	require.Equal(t, "ARIN", r.Registry)
}

func TestLookupIPSpecialUse(t *testing.T) {
	c := NewClient(failDoer{t}, time.Minute) // any network call fails the test
	// Documentation (TEST-NET-1/2/3), CGNAT, reserved, this-network, and
	// IPv6 documentation ranges are not globally routed and must short-circuit.
	for _, ip := range []string{
		"192.0.2.1", "198.51.100.5", "203.0.113.9",
		"100.64.0.1", "240.0.0.1", "0.1.2.3", "2001:db8::1",
	} {
		r, err := c.LookupIP(context.Background(), ip)
		require.NoError(t, err, ip)
		require.NotEmpty(t, r.RoutingNote, ip)
		require.Empty(t, r.OriginASN, ip)
	}
}
