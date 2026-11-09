package lookup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// lgBody is a trimmed real RIPEstat looking-glass response (field names copied
// verbatim from a live stat.ripe.net fetch 2026-11-09): two collectors, the
// second with two peers sharing an origin.
const lgBody = `{
  "data": {
    "rrcs": [
      {
        "rrc": "RRC01",
        "location": "London, GB",
        "scope": "LINX / LONAP",
        "peers": [
          {
            "asn_origin": "1205",
            "as_path": "34854 2914 1853 1205",
            "community": "2914:410",
            "next_hop": "195.66.224.138",
            "origin": "IGP",
            "peer": "195.66.224.138",
            "prefix": "140.78.0.0/16",
            "last_updated": "2026-10-27T20:35:33"
          }
        ]
      },
      {
        "rrc": "RRC00",
        "location": "Amsterdam, NL",
        "scope": "RIPE NCC",
        "peers": [
          {
            "asn_origin": "1205",
            "as_path": "15692 2914 1764 1853 1205",
            "community": "174:70 2914:410",
            "next_hop": "5.57.80.113",
            "origin": "IGP",
            "peer": "5.57.80.113",
            "prefix": "140.78.0.0/16",
            "last_updated": "2026-10-27T20:35:33"
          },
          {
            "asn_origin": "1205",
            "as_path": "",
            "next_hop": "5.57.80.114",
            "origin": "IGP",
            "peer": "5.57.80.114",
            "prefix": "140.78.0.0/16",
            "last_updated": "2026-10-27T20:35:33"
          }
        ]
      }
    ]
  }
}`

func TestRipeLookingGlass(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{"/looking-glass/": {body: lgBody}})
	cols, err := c.ripeLookingGlass(context.Background(), "140.78.0.0/16")
	require.NoError(t, err)
	require.Len(t, cols, 2)

	require.Equal(t, "RRC01", cols[0].RRC)
	require.Equal(t, "London, GB", cols[0].Location)
	require.Equal(t, "LINX / LONAP", cols[0].Scope)
	require.Len(t, cols[0].Peers, 1)

	p := cols[1].Peers[0]
	require.Equal(t, []string{"15692", "2914", "1764", "1853", "1205"}, p.ASPath)
	require.Equal(t, "1205", p.OriginASN)
	require.Equal(t, "5.57.80.113", p.NextHop)
	require.Equal(t, "IGP", p.Origin)
	require.Equal(t, "140.78.0.0/16", p.Prefix)

	// An empty as_path yields an empty slice, not a one-element [""].
	require.Empty(t, cols[1].Peers[1].ASPath)
}
