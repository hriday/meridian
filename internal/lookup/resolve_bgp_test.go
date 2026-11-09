package lookup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// peerJSON builds one looking-glass peer object with the given origin and path.
func rpkiValidBody(prefix string) string {
	return `{"data":{"status":"valid","validating_roas":[{"prefix":"` + prefix + `","max_length":16,"origin":"1205"}]}}`
}

// lgAggBody: 4 peers across 2 collectors — three share one path, one differs.
const lgAggBody = `{"data":{"rrcs":[
  {"rrc":"RRC01","location":"London, GB","scope":"LINX","peers":[
    {"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"},
    {"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"},
    {"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"}
  ]},
  {"rrc":"RRC00","location":"Amsterdam, NL","scope":"RIPE NCC","peers":[
    {"asn_origin":"1205","as_path":"3356 1205","prefix":"140.78.0.0/16"}
  ]}
]}}`

func TestLookupBGPAggregatesPaths(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"/looking-glass/":   {body: lgAggBody},
		"/rpki-validation/": {body: rpkiValidBody("140.78.0.0/16")},
	})
	r, err := c.LookupBGP(context.Background(), "140.78.0.0/16")
	require.NoError(t, err)
	require.Equal(t, 2, r.Collectors)
	require.Equal(t, 4, r.Peers)
	require.Equal(t, "140.78.0.0/16", r.Prefix)
	require.Equal(t, []string{"1205"}, r.OriginASNs)
	require.False(t, r.MOAS)
	require.Equal(t, 3, r.Paths[0].Peers)
	require.Equal(t, 75, r.Paths[0].Percent)
	require.Equal(t, []string{"174", "1205"}, r.Paths[0].Path)
	require.Equal(t, "valid", r.RPKIState)
	require.Empty(t, r.RoutingSrc.Err)
}

// lgBucketBody: a 4-hop path, a 2-hop path, and an origin-only path.
const lgBucketBody = `{"data":{"rrcs":[
  {"rrc":"RRC01","location":"London","scope":"LINX","peers":[
    {"asn_origin":"1205","as_path":"3356 2914 1764 1205","prefix":"140.78.0.0/16"},
    {"asn_origin":"1205","as_path":"6939 1205","prefix":"140.78.0.0/16"},
    {"asn_origin":"1205","as_path":"1205","prefix":"140.78.0.0/16"}
  ]}
]}}`

func TestLookupBGPPathBuckets(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"/looking-glass/":   {body: lgBucketBody},
		"/rpki-validation/": {body: rpkiValidBody("140.78.0.0/16")},
	})
	r, err := c.LookupBGP(context.Background(), "140.78.0.0/16")
	require.NoError(t, err)
	// Upstream = AS directly before the origin on a path.
	require.Contains(t, r.Upstreams, "1764")
	require.Contains(t, r.Upstreams, "6939")
	// Transit = intermediate ASes that aren't upstreams or the origin.
	require.Contains(t, r.Transit, "3356")
	require.Contains(t, r.Transit, "2914")
	// The origin never appears in upstreams or transit.
	require.NotContains(t, r.Upstreams, "1205")
	require.NotContains(t, r.Transit, "1205")
}

// lgMOASBody: two distinct origins for the same prefix.
const lgMOASBody = `{"data":{"rrcs":[
  {"rrc":"RRC01","location":"London","scope":"LINX","peers":[
    {"asn_origin":"1205","as_path":"174 1205","prefix":"140.78.0.0/16"},
    {"asn_origin":"64500","as_path":"3356 64500","prefix":"140.78.0.0/16"}
  ]}
]}}`

func TestLookupBGPMOAS(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"/looking-glass/":   {body: lgMOASBody},
		"/rpki-validation/": {body: rpkiValidBody("140.78.0.0/16")},
	})
	r, err := c.LookupBGP(context.Background(), "140.78.0.0/16")
	require.NoError(t, err)
	require.Len(t, r.OriginASNs, 2)
	require.True(t, r.MOAS)
	require.Equal(t, "valid", r.RPKIState) // queried for the first origin
}

func TestLookupBGPNotSeen(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"/looking-glass/": {body: `{"data":{"rrcs":[]}}`},
	})
	r, err := c.LookupBGP(context.Background(), "192.0.2.0/24")
	require.NoError(t, err)
	require.Equal(t, 0, r.Collectors)
	require.Equal(t, 0, r.Peers)
	require.Empty(t, r.Paths)
	require.Empty(t, r.RPKISrc.Label) // no RPKI call when there's no origin
}

func TestLookupBGPRPKIDegrades(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"/looking-glass/":   {body: lgAggBody},
		"/rpki-validation/": {code: 500, body: "boom"},
	})
	r, err := c.LookupBGP(context.Background(), "140.78.0.0/16")
	require.NoError(t, err)
	require.NotEmpty(t, r.Paths)       // routing still present
	require.NotEmpty(t, r.RPKISrc.Err) // RPKI degraded
	require.Empty(t, r.RPKIState)
}
