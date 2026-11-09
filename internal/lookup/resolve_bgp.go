package lookup

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"
)

// BGPPath is one distinct AS path and how many peers observe it. Path reads
// left→right as seen→origin; ASNs are plain decimal.
type BGPPath struct {
	Path    []string `json:"path"`
	Peers   int      `json:"peers"`
	Percent int      `json:"percent"`
}

// BGPCollector is one RIS route collector that sees the resource.
type BGPCollector struct {
	RRC      string `json:"rrc"`
	Location string `json:"location"`
	Scope    string `json:"scope"`
	Peers    int    `json:"peers"`
}

// BGPReport is the control-plane view of a prefix/IP: who originates it, how
// widely it's seen, the AS paths to it, its upstreams/transit, RPKI validity,
// and the collectors (vantages) that observe it. ASN tokens are plain decimal
// throughout; the UI prepends "AS" for display.
type BGPReport struct {
	Input      string         `json:"input"`
	Prefix     string         `json:"prefix"`
	OriginASNs []string       `json:"originAsns"`
	MOAS       bool           `json:"moas"`
	Collectors int            `json:"collectors"`
	Peers      int            `json:"peers"`
	Paths      []BGPPath      `json:"paths"`
	Upstreams  []string       `json:"upstreams"`
	Transit    []string       `json:"transit"`
	RPKIState  string         `json:"rpkiState"`
	Vantages   []BGPCollector `json:"vantages"`
	RoutingSrc Source         `json:"routingSrc"`
	RPKISrc    Source         `json:"rpkiSrc"`
}

// LookupBGP fetches the RIPEstat looking-glass view of input (a prefix or IP)
// and aggregates it: origin AS(es), ranked AS paths, upstream/transit sets,
// visibility counts, the collectors that see it, and RPKI validity of the
// (origin, prefix) pair. A failed source degrades its card, not the report.
func (c *Client) LookupBGP(ctx context.Context, input string) (BGPReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	r := BGPReport{Input: input}

	r.RoutingSrc = Source{Label: "RIPEstat looking-glass", FetchedAt: now()}
	cols, err := c.ripeLookingGlass(ctx, input)
	if err != nil {
		r.RoutingSrc.Err = err.Error()
		return r, nil
	}

	originSet := map[string]bool{}
	prefixCount := map[string]int{}
	pathCount := map[string]int{} // joined path → peer count
	pathOf := map[string][]string{}
	var totalPeers int

	for _, col := range cols {
		if len(col.Peers) == 0 {
			continue
		}
		r.Collectors++
		r.Vantages = append(r.Vantages, BGPCollector{
			RRC: col.RRC, Location: col.Location, Scope: col.Scope, Peers: len(col.Peers),
		})
		for _, p := range col.Peers {
			totalPeers++
			if p.OriginASN != "" {
				originSet[p.OriginASN] = true
			}
			if p.Prefix != "" {
				prefixCount[p.Prefix]++
			}
			// A peer still counts toward visibility even with no path, but an
			// empty path is not rendered as a blank row.
			if len(p.ASPath) == 0 {
				continue
			}
			key := strings.Join(p.ASPath, " ")
			pathCount[key]++
			if _, ok := pathOf[key]; !ok {
				pathOf[key] = p.ASPath
			}
		}
	}
	r.Peers = totalPeers

	r.OriginASNs = sortedKeys(originSet)
	r.MOAS = len(r.OriginASNs) > 1
	r.Prefix = mostCommon(prefixCount)

	// Ranked distinct paths.
	for key, n := range pathCount {
		pct := 0
		if totalPeers > 0 {
			pct = int(math.Round(float64(n) / float64(totalPeers) * 100))
		}
		r.Paths = append(r.Paths, BGPPath{Path: pathOf[key], Peers: n, Percent: pct})
	}
	sort.Slice(r.Paths, func(i, j int) bool {
		if r.Paths[i].Peers != r.Paths[j].Peers {
			return r.Paths[i].Peers > r.Paths[j].Peers
		}
		return strings.Join(r.Paths[i].Path, " ") < strings.Join(r.Paths[j].Path, " ")
	})

	// Upstream/transit buckets (disjoint, excluding origins).
	upstreamSet := map[string]bool{}
	for key := range pathCount {
		path := pathOf[key]
		if len(path) >= 2 {
			up := path[len(path)-2]
			if !originSet[up] {
				upstreamSet[up] = true
			}
		}
	}
	transitSet := map[string]bool{}
	for key := range pathCount {
		path := pathOf[key]
		// Positions before the upstream: indices 0..len-3.
		for i := 0; i+2 < len(path); i++ {
			as := path[i]
			if !originSet[as] && !upstreamSet[as] {
				transitSet[as] = true
			}
		}
	}
	r.Upstreams = sortedKeys(upstreamSet)
	r.Transit = sortedKeys(transitSet)

	sort.Slice(r.Vantages, func(i, j int) bool {
		if r.Vantages[i].Peers != r.Vantages[j].Peers {
			return r.Vantages[i].Peers > r.Vantages[j].Peers
		}
		return r.Vantages[i].RRC < r.Vantages[j].RRC
	})

	// RPKI validity of the primary origin against the matched prefix.
	if len(r.OriginASNs) >= 1 && r.Prefix != "" {
		r.RPKISrc = Source{Label: "RIPEstat RPKI", FetchedAt: now()}
		if state, _, err := c.ripeRPKI(ctx, r.Prefix, r.OriginASNs[0]); err != nil {
			r.RPKISrc.Err = err.Error()
		} else {
			r.RPKIState = state
		}
	}
	return r, nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func mostCommon(m map[string]int) string {
	best, bestN := "", 0
	for k, n := range m {
		if n > bestN || (n == bestN && k < best) {
			best, bestN = k, n
		}
	}
	return best
}
