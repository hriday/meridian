// Package dnsprop checks a DNS record across many global public resolvers at
// once and reports whether they agree. After you change a record, this shows
// how far the change has propagated; a split answer means some resolvers still
// serve the old value (or the zone is inconsistent / split-horizon).
//
// It is keyless and point-in-time: it queries live, reports, and keeps nothing.
package dnsprop

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/hriday/meridian/internal/dnsx"
)

// Resolver is a named public DNS resolver to query.
type Resolver struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Loc  string `json:"loc,omitempty"`
}

// QueryFunc is the per-resolver query seam; dnsx.Query satisfies it.
type QueryFunc func(ctx context.Context, name, rtype, server string) (dnsx.Result, error)

// Answer is one resolver's outcome.
type Answer struct {
	Resolver  Resolver `json:"resolver"`
	Values    []string `json:"values"`
	TTL       uint32   `json:"ttl"`
	Rcode     string   `json:"rcode"`
	RTTMillis int      `json:"rttMillis"`
	Err       string   `json:"err,omitempty"`
	Group     int      `json:"group"` // index into Report.Groups, -1 if errored
}

// Group is a distinct answer and the resolvers that returned it.
type Group struct {
	Values    []string `json:"values"`
	Rcode     string   `json:"rcode"`
	Resolvers []string `json:"resolvers"`
	Count     int      `json:"count"`
}

// Report is the whole picture for one name/type across all resolvers.
type Report struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Answers    []Answer `json:"answers"`
	Groups     []Group  `json:"groups"`
	Consensus  bool     `json:"consensus"`  // every resolver that answered agrees
	Propagated int      `json:"propagated"` // resolvers in the largest group
	Total      int      `json:"total"`      // resolvers that answered (no transport error)
}

// Check queries every resolver concurrently for name's rtype record and groups
// the answers. Transport errors are reported per resolver but excluded from the
// consensus math.
func Check(ctx context.Context, q QueryFunc, name, rtype string, resolvers []Resolver) Report {
	if rtype == "" {
		rtype = "A"
	}
	answers := make([]Answer, len(resolvers))
	var wg sync.WaitGroup
	for i, r := range resolvers {
		wg.Add(1)
		go func(i int, r Resolver) {
			defer wg.Done()
			a := Answer{Resolver: r, Group: -1}
			res, err := q(ctx, name, rtype, r.IP)
			if err != nil {
				a.Err = err.Error()
				answers[i] = a
				return
			}
			recs := ofType(res.Answers, rtype)
			a.Rcode = res.Rcode
			a.RTTMillis = res.RTTMillis
			a.TTL = minTTL(recs)
			a.Values = normalize(recs)
			answers[i] = a
		}(i, r)
	}
	wg.Wait()

	rep := Report{Name: name, Type: strings.ToUpper(rtype), Answers: answers}
	groupKey := func(a Answer) string {
		return a.Rcode + "|" + strings.Join(a.Values, ",")
	}

	idx := map[string]int{}
	for ai, a := range answers {
		if a.Err != "" {
			continue
		}
		rep.Total++
		key := groupKey(a)
		gi, ok := idx[key]
		if !ok {
			gi = len(rep.Groups)
			idx[key] = gi
			rep.Groups = append(rep.Groups, Group{Values: a.Values, Rcode: a.Rcode})
		}
		rep.Groups[gi].Resolvers = append(rep.Groups[gi].Resolvers, a.Resolver.Name)
		rep.Groups[gi].Count++
		rep.Answers[ai].Group = gi
	}

	// Largest group first; ties broken by key for a stable order. Remember each
	// group's original index so Answer.Group stays correct after the sort.
	order := make([]int, len(rep.Groups))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(x, y int) bool {
		gx, gy := rep.Groups[order[x]], rep.Groups[order[y]]
		if gx.Count != gy.Count {
			return gx.Count > gy.Count
		}
		return strings.Join(gx.Values, ",") < strings.Join(gy.Values, ",")
	})
	remap := make([]int, len(rep.Groups))
	sorted := make([]Group, len(rep.Groups))
	for newIdx, oldIdx := range order {
		sorted[newIdx] = rep.Groups[oldIdx]
		remap[oldIdx] = newIdx
	}
	rep.Groups = sorted
	for i := range rep.Answers {
		if rep.Answers[i].Group >= 0 {
			rep.Answers[i].Group = remap[rep.Answers[i].Group]
		}
	}

	rep.Consensus = len(rep.Groups) == 1
	if len(rep.Groups) > 0 {
		rep.Propagated = rep.Groups[0].Count
	}
	return rep
}

// ofType keeps only records of the requested type, dropping DNSSEC RRSIGs and
// any incidental records (e.g. a CNAME returned alongside the A it resolves to)
// so propagation compares like with like. An unknown/empty type keeps all.
func ofType(recs []dnsx.Record, rtype string) []dnsx.Record {
	want := strings.ToUpper(strings.TrimSpace(rtype))
	if want == "" || want == "ANY" {
		return recs
	}
	out := make([]dnsx.Record, 0, len(recs))
	for _, r := range recs {
		if strings.EqualFold(r.Type, want) {
			out = append(out, r)
		}
	}
	return out
}

// normalize returns the record values, lowercased and sorted, so two resolvers
// that return the same set in a different order compare equal.
func normalize(recs []dnsx.Record) []string {
	vals := make([]string, 0, len(recs))
	for _, r := range recs {
		v := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(r.Value)), ".")
		vals = append(vals, v)
	}
	sort.Strings(vals)
	return vals
}

func minTTL(recs []dnsx.Record) uint32 {
	var min uint32
	for i, r := range recs {
		if i == 0 || r.TTL < min {
			min = r.TTL
		}
	}
	return min
}
