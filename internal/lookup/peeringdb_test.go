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

// pdbServer serves PeeringDB fixtures chosen by path fragment and points
// peeringDBBase at the test server. The netixlan/netfac paths both contain
// "net", so they are matched before the bare "net" fragment.
func pdbServer(t *testing.T, fixtures map[string]srvResp) *Client {
	order := []string{"netixlan", "netfac", "net"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, frag := range order {
			f, ok := fixtures[frag]
			if !ok || !strings.Contains(r.URL.Path, frag) {
				continue
			}
			if f.code != 0 {
				w.WriteHeader(f.code)
			}
			_, _ = w.Write([]byte(f.body))
			return
		}
		http.Error(w, "no fixture: "+r.URL.Path, 404)
	}))
	t.Cleanup(srv.Close)
	old := peeringDBBase
	peeringDBBase = srv.URL + "/"
	t.Cleanup(func() { peeringDBBase = old })
	return NewClient(http.DefaultClient, time.Minute)
}

func TestPeeringDBParsesNetAndLists(t *testing.T) {
	c := pdbServer(t, map[string]srvResp{
		"net":      {body: `{"data":[{"id":4224,"name":"Cloudflare","info_type":"Content","info_scope":"Global","info_ratio":"Mostly Outbound","policy_general":"Open","irr_as_set":"ARIN::AS13335:AS-CLOUDFLARE","website":"https://www.cloudflare.com","info_prefixes4":80000,"info_prefixes6":5000,"ix_count":2,"fac_count":1}]}`},
		"netixlan": {body: `{"data":[{"name":"Equinix Paris","speed":100000},{"name":"Equinix Paris","speed":200000},{"name":"AMS-IX","speed":400000}]}`},
		"netfac":   {body: `{"data":[{"name":"Equinix DC1","city":"Ashburn","country":"US"}]}`},
	})
	p, err := c.peeringDB(context.Background(), "AS13335")
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, "Cloudflare", p.Name)
	require.Equal(t, "Content", p.Type)
	require.Equal(t, "Global", p.Scope)
	require.Equal(t, "Mostly Outbound", p.Ratio)
	require.Equal(t, "Open", p.Policy)
	require.Equal(t, "ARIN::AS13335:AS-CLOUDFLARE", p.IRRASSet)
	require.Equal(t, 80000, p.V4Prefixes)
	require.Equal(t, 5000, p.V6Prefixes)
	require.Equal(t, 2, p.IXCount)
	require.Equal(t, 1, p.FacCount)
	// dedupe by IX name keeping max speed, sorted by speed desc
	require.Equal(t, []PeerIX{{Name: "AMS-IX", Speed: 400000}, {Name: "Equinix Paris", Speed: 200000}}, p.IXPs)
	require.Equal(t, []PeerFac{{Name: "Equinix DC1", City: "Ashburn", Country: "US"}}, p.Facilities)
}

func TestPeeringDBNotInRegistry(t *testing.T) {
	c := pdbServer(t, map[string]srvResp{"net": {body: `{"data":[]}`}})
	p, err := c.peeringDB(context.Background(), "AS65000")
	require.NoError(t, err)
	require.Nil(t, p)
}

func TestPeeringDBListErrorDegrades(t *testing.T) {
	c := pdbServer(t, map[string]srvResp{
		"net":      {body: `{"data":[{"id":1,"name":"N","info_type":"NSP","ix_count":5}]}`},
		"netixlan": {code: 500, body: ``},
		"netfac":   {code: 404, body: ``},
	})
	p, err := c.peeringDB(context.Background(), "1")
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, "N", p.Name)
	require.Empty(t, p.IXPs)
	require.Empty(t, p.Facilities)
}

func TestPeeringDBNetError(t *testing.T) {
	c := pdbServer(t, map[string]srvResp{"net": {code: 500, body: ``}})
	p, err := c.peeringDB(context.Background(), "AS13335")
	require.Error(t, err)
	require.Nil(t, p)
}
