package lookup

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// peeringDBBase is the PeeringDB public API root (keyless). It is a package var
// so tests can point it at an httptest server.
var peeringDBBase = "https://www.peeringdb.com/api/"

// PeerIX is one Internet Exchange a network is present at. Speed is the maximum
// observed port speed (Mbps) across the network's ports at that exchange.
type PeerIX struct {
	Name  string `json:"name"`
	Speed int    `json:"speed"`
}

// PeerFac is one facility (data center) a network is present in.
type PeerFac struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
}

// Peering is a network's PeeringDB record: how it peers, where it's present.
type Peering struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`     // content / nsp / ...
	Scope      string    `json:"scope"`    // global / regional / ...
	Ratio      string    `json:"ratio"`    // mostly outbound / balanced / ...
	Policy     string    `json:"policy"`   // open / selective / restrictive
	IRRASSet   string    `json:"irrAsSet"` // IRR AS-SET macro
	Website    string    `json:"website"`
	V4Prefixes int       `json:"v4Prefixes"` // announced-prefix estimate (v4)
	V6Prefixes int       `json:"v6Prefixes"` // announced-prefix estimate (v6)
	IXCount    int       `json:"ixCount"`    // total IX presences
	FacCount   int       `json:"facCount"`   // total facility presences
	IXPs       []PeerIX  `json:"ixps"`       // all IX presences, deduped by name
	Facilities []PeerFac `json:"facilities"` // all facility presences
}

// netRec mirrors a PeeringDB net record (fields copied from the live API).
type netRec struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	InfoType      string `json:"info_type"`
	InfoScope     string `json:"info_scope"`
	InfoRatio     string `json:"info_ratio"`
	PolicyGeneral string `json:"policy_general"`
	IRRASSet      string `json:"irr_as_set"`
	Website       string `json:"website"`
	InfoPrefixes4 int    `json:"info_prefixes4"`
	InfoPrefixes6 int    `json:"info_prefixes6"`
	IXCount       int    `json:"ix_count"`
	FacCount      int    `json:"fac_count"`
}

type netixlanRec struct {
	Name  string `json:"name"`
	Speed int    `json:"speed"`
}

type netfacRec struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Country string `json:"country"`
}

// peeringDB fetches a network's PeeringDB enrichment for asn ("13335" or
// "AS13335"). It returns (nil, nil) when the ASN isn't in PeeringDB, (nil, err)
// when the primary net fetch fails, and a populated *Peering otherwise. The IX
// and facility list fetches are best-effort: an error on either leaves that
// list empty rather than failing the whole lookup.
func (c *Client) peeringDB(ctx context.Context, asn string) (*Peering, error) {
	num := strings.TrimLeft(strings.ToUpper(strings.TrimSpace(asn)), "AS")

	var net struct {
		Data []netRec `json:"data"`
	}
	if err := c.GetJSON(ctx, peeringDBBase+"net?asn="+num, &net); err != nil {
		return nil, err
	}
	if len(net.Data) == 0 {
		return nil, nil
	}
	n := net.Data[0]
	p := &Peering{
		Name:       n.Name,
		Type:       n.InfoType,
		Scope:      n.InfoScope,
		Ratio:      n.InfoRatio,
		Policy:     n.PolicyGeneral,
		IRRASSet:   n.IRRASSet,
		Website:    n.Website,
		V4Prefixes: n.InfoPrefixes4,
		V6Prefixes: n.InfoPrefixes6,
		IXCount:    n.IXCount,
		FacCount:   n.FacCount,
	}
	p.IXPs = c.peeringIXPs(ctx, n.ID)
	p.Facilities = c.peeringFacilities(ctx, n.ID)
	return p, nil
}

// peeringIXPs fetches the net's IX presences, deduped by IX name keeping the max
// observed speed, sorted by speed descending then name. Errors yield nil.
func (c *Client) peeringIXPs(ctx context.Context, netID int) []PeerIX {
	var res struct {
		Data []netixlanRec `json:"data"`
	}
	if err := c.GetJSON(ctx, fmt.Sprintf("%snetixlan?net_id=%d", peeringDBBase, netID), &res); err != nil {
		return nil
	}
	maxSpeed := map[string]int{}
	for _, r := range res.Data {
		if r.Name == "" {
			continue
		}
		if cur, seen := maxSpeed[r.Name]; !seen || r.Speed > cur {
			maxSpeed[r.Name] = r.Speed
		}
	}
	out := make([]PeerIX, 0, len(maxSpeed))
	for name, speed := range maxSpeed {
		out = append(out, PeerIX{Name: name, Speed: speed})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Speed != out[j].Speed {
			return out[i].Speed > out[j].Speed
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// peeringFacilities fetches the net's facility presences, sorted by name then
// city. Errors yield nil.
func (c *Client) peeringFacilities(ctx context.Context, netID int) []PeerFac {
	var res struct {
		Data []netfacRec `json:"data"`
	}
	if err := c.GetJSON(ctx, fmt.Sprintf("%snetfac?net_id=%d", peeringDBBase, netID), &res); err != nil {
		return nil
	}
	out := make([]PeerFac, 0, len(res.Data))
	for _, r := range res.Data {
		if r.Name == "" {
			continue
		}
		out = append(out, PeerFac{Name: r.Name, City: r.City, Country: r.Country})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].City < out[j].City
	})
	return out
}
