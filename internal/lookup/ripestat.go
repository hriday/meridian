package lookup

import (
	"context"
	"fmt"
	"net/url"
)

// ripeBase is the RIPEstat data API root. It is a package var so tests can
// point it at an httptest server.
var ripeBase = "https://stat.ripe.net/data/"

func ripeURL(call, resource string) string {
	return fmt.Sprintf("%s%s/data.json?resource=%s", ripeBase, call, url.QueryEscape(resource))
}

// ripeNetworkInfo returns the covering prefix and origin ASN(s) RIPEstat sees
// for an IP. An unannounced address yields an empty prefix and no ASNs.
func (c *Client) ripeNetworkInfo(ctx context.Context, ip string) (prefix string, asns []string, err error) {
	var r struct {
		Data struct {
			Prefix string   `json:"prefix"`
			ASNs   []string `json:"asns"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, ripeURL("network-info", ip), &r); err != nil {
		return "", nil, err
	}
	return r.Data.Prefix, r.Data.ASNs, nil
}

// ripeASOverview returns the holder/name RIPEstat records for an ASN.
func (c *Client) ripeASOverview(ctx context.Context, asn string) (holder string, err error) {
	var r struct {
		Data struct {
			Holder string `json:"holder"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, ripeURL("as-overview", asn), &r); err != nil {
		return "", err
	}
	return r.Data.Holder, nil
}

// ripeRoutingStatus returns the counts of announced IPv4/IPv6 prefixes and the
// number of observed BGP neighbours for an ASN.
func (c *Client) ripeRoutingStatus(ctx context.Context, asn string) (v4Count, v6Count, neighbours int, err error) {
	var r struct {
		Data struct {
			AnnouncedSpace struct {
				V4 struct {
					Prefixes int `json:"prefixes"`
				} `json:"v4"`
				V6 struct {
					Prefixes int `json:"prefixes"`
				} `json:"v6"`
			} `json:"announced_space"`
			ObservedNeighbours int `json:"observed_neighbours"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, ripeURL("routing-status", asn), &r); err != nil {
		return 0, 0, 0, err
	}
	d := r.Data
	return d.AnnouncedSpace.V4.Prefixes, d.AnnouncedSpace.V6.Prefixes, d.ObservedNeighbours, nil
}
