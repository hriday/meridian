package lookup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupPrefixCombines(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"prefix-overview":  {body: `{"data":{"announced":true,"asns":[{"asn":13335,"holder":"CLOUDFLARENET, US"}]}}`},
		"related-prefixes": {body: `{"data":{"prefixes":[{"prefix":"104.16.0.0/12","relationship":"less-specific"},{"prefix":"104.16.123.0/24","relationship":"more-specific"}]}}`},
		"rpki-validation":  {body: `{"data":{"status":"valid","validating_roas":[{"prefix":"104.16.0.0/13","max_length":24,"origin":"13335"}]}}`},
		"/ip/":             {body: arinIPBody},
	})
	r, err := c.LookupPrefix(context.Background(), "104.16.0.0/13")
	require.NoError(t, err)
	require.Equal(t, "AS13335", r.OriginASN)
	require.True(t, r.Announced)
	require.Equal(t, "valid", r.RPKIState)
	require.Equal(t, []string{"104.16.0.0/12"}, r.LessSpecific)
	require.Equal(t, []string{"104.16.123.0/24"}, r.MoreSpecific)
	require.Equal(t, "Cloudflare, Inc.", r.RegisteredOrg)
	require.Empty(t, r.RoutingSrc.Err)
}

func TestLookupPrefixUnannounced(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"prefix-overview":  {body: `{"data":{"announced":false,"asns":[]}}`},
		"related-prefixes": {body: `{"data":{"prefixes":[]}}`},
		"/ip/":             {body: arinIPBody},
	})
	r, err := c.LookupPrefix(context.Background(), "192.0.2.0/24")
	require.NoError(t, err)
	require.Empty(t, r.OriginASN)
	require.False(t, r.Announced)
}

func TestLookupIPHasRPKI(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{
		"network-info":    {body: `{"data":{"prefix":"104.16.0.0/13","asns":["13335"]}}`},
		"as-overview":     {body: `{"data":{"holder":"CLOUDFLARENET, US"}}`},
		"rpki-validation": {body: `{"data":{"status":"valid","validating_roas":[{"prefix":"104.16.0.0/13","max_length":24,"origin":"13335"}]}}`},
		"/ip/":            {body: arinIPBody},
	})
	r, err := c.LookupIP(context.Background(), "104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "valid", r.RPKIState)
}
