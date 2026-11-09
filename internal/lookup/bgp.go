package lookup

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// lgPeer is one BGP peer's view of the queried resource at a route collector.
// ASPath reads left→right as seen→origin: the last element is the origin AS.
type lgPeer struct {
	OriginASN   string   // asn_origin, plain decimal
	ASPath      []string // as_path split on whitespace
	NextHop     string
	Origin      string // IGP/EGP/…
	Peer        string
	Prefix      string
	LastUpdated string
}

// lgCollector is one RIS route collector (RRC) and the peers at it that see
// the queried resource.
type lgCollector struct {
	RRC      string
	Location string
	Scope    string
	Peers    []lgPeer
}

// ripeLookingGlass fetches RIPEstat's aggregated looking-glass view of a prefix
// or IP: the RIS route collectors that see it and, per collector, each peer's
// AS path, origin, and next-hop.
func (c *Client) ripeLookingGlass(ctx context.Context, resource string) ([]lgCollector, error) {
	u := fmt.Sprintf("%slooking-glass/data.json?resource=%s", ripeBase, url.QueryEscape(resource))
	var r struct {
		Data struct {
			RRCs []struct {
				RRC      string `json:"rrc"`
				Location string `json:"location"`
				Scope    string `json:"scope"`
				Peers    []struct {
					OriginASN   string `json:"asn_origin"`
					ASPath      string `json:"as_path"`
					NextHop     string `json:"next_hop"`
					Origin      string `json:"origin"`
					Peer        string `json:"peer"`
					Prefix      string `json:"prefix"`
					LastUpdated string `json:"last_updated"`
				} `json:"peers"`
			} `json:"rrcs"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, u, &r); err != nil {
		return nil, err
	}

	cols := make([]lgCollector, 0, len(r.Data.RRCs))
	for _, rrc := range r.Data.RRCs {
		col := lgCollector{RRC: rrc.RRC, Location: rrc.Location, Scope: rrc.Scope}
		for _, p := range rrc.Peers {
			col.Peers = append(col.Peers, lgPeer{
				OriginASN:   p.OriginASN,
				ASPath:      strings.Fields(p.ASPath),
				NextHop:     p.NextHop,
				Origin:      p.Origin,
				Peer:        p.Peer,
				Prefix:      p.Prefix,
				LastUpdated: p.LastUpdated,
			})
		}
		cols = append(cols, col)
	}
	return cols, nil
}
