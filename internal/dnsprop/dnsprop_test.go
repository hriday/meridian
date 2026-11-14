package dnsprop

import (
	"context"
	"errors"
	"testing"

	"github.com/hriday/meridian/internal/dnsx"
)

// scripted returns a QueryFunc answering from a map of resolver IP -> result,
// and errs for IPs in the errSet.
func scripted(byIP map[string]dnsx.Result, errIPs ...string) QueryFunc {
	errSet := map[string]bool{}
	for _, ip := range errIPs {
		errSet[ip] = true
	}
	return func(ctx context.Context, name, rtype, server string) (dnsx.Result, error) {
		if errSet[server] {
			return dnsx.Result{}, errors.New("timeout")
		}
		return byIP[server], nil
	}
}

func rec(v string) dnsx.Record { return dnsx.Record{Type: "A", Value: v, TTL: 300} }
func recT(typ, v string) dnsx.Record {
	return dnsx.Record{Type: typ, Value: v, TTL: 300}
}

func TestCheckFiltersToRequestedType(t *testing.T) {
	// A resolver that also returns DNSSEC RRSIG / a CNAME alongside the A record
	// must compare only on the A value, or RRSIG noise causes false divergence.
	resolvers := []Resolver{{Name: "A", IP: "1.1.1.1"}, {Name: "B", IP: "8.8.8.8"}}
	q := scripted(map[string]dnsx.Result{
		"1.1.1.1": {Rcode: "NOERROR", Answers: []dnsx.Record{recT("A", "203.0.113.5"), recT("RRSIG", "sig-one")}},
		"8.8.8.8": {Rcode: "NOERROR", Answers: []dnsx.Record{recT("A", "203.0.113.5"), recT("RRSIG", "sig-two-different")}},
	})
	rep := Check(context.Background(), q, "example.com", "A", resolvers)
	if !rep.Consensus {
		t.Errorf("A records match; differing RRSIGs must not break consensus; groups=%d", len(rep.Groups))
	}
	if len(rep.Groups) == 1 && len(rep.Groups[0].Values) != 1 {
		t.Errorf("group should hold only the A value, got %v", rep.Groups[0].Values)
	}
}

func TestCheckConsensus(t *testing.T) {
	resolvers := []Resolver{{Name: "A", IP: "1.1.1.1"}, {Name: "B", IP: "8.8.8.8"}}
	q := scripted(map[string]dnsx.Result{
		"1.1.1.1": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("203.0.113.5")}},
		"8.8.8.8": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("203.0.113.5")}},
	})
	rep := Check(context.Background(), q, "example.com", "A", resolvers)
	if !rep.Consensus {
		t.Errorf("identical answers should be consensus; groups=%d", len(rep.Groups))
	}
	if rep.Total != 2 || rep.Propagated != 2 {
		t.Errorf("Total=%d Propagated=%d, want 2/2", rep.Total, rep.Propagated)
	}
	if len(rep.Groups) != 1 {
		t.Fatalf("want 1 group, got %d", len(rep.Groups))
	}
}

func TestCheckDivergenceOrdersGroupsBySize(t *testing.T) {
	resolvers := []Resolver{
		{Name: "A", IP: "1.1.1.1"}, {Name: "B", IP: "8.8.8.8"}, {Name: "C", IP: "9.9.9.9"},
	}
	// Two resolvers on the new IP, one still on the old.
	q := scripted(map[string]dnsx.Result{
		"1.1.1.1": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("203.0.113.9")}}, // new
		"8.8.8.8": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("203.0.113.9")}}, // new
		"9.9.9.9": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("203.0.113.1")}}, // old
	})
	rep := Check(context.Background(), q, "example.com", "A", resolvers)
	if rep.Consensus {
		t.Error("split answers should not be consensus")
	}
	if len(rep.Groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(rep.Groups))
	}
	if rep.Groups[0].Count != 2 || rep.Groups[1].Count != 1 {
		t.Errorf("groups should be largest-first: got %d then %d", rep.Groups[0].Count, rep.Groups[1].Count)
	}
	if rep.Propagated != 2 {
		t.Errorf("Propagated=%d, want 2 (the majority)", rep.Propagated)
	}
	// Every non-errored answer must point at a valid group index.
	for _, a := range rep.Answers {
		if a.Group < 0 || a.Group >= len(rep.Groups) {
			t.Errorf("answer %s has bad group index %d", a.Resolver.Name, a.Group)
		}
		if rep.Groups[a.Group].Count == 0 {
			t.Errorf("answer %s mapped to empty group", a.Resolver.Name)
		}
	}
}

func TestCheckExcludesTransportErrors(t *testing.T) {
	resolvers := []Resolver{{Name: "A", IP: "1.1.1.1"}, {Name: "Dead", IP: "10.0.0.1"}}
	q := scripted(map[string]dnsx.Result{
		"1.1.1.1": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("203.0.113.5")}},
	}, "10.0.0.1")
	rep := Check(context.Background(), q, "example.com", "A", resolvers)
	if rep.Total != 1 {
		t.Errorf("Total=%d, want 1 (the dead resolver is excluded)", rep.Total)
	}
	if !rep.Consensus {
		t.Error("one good answer with one dead resolver is still consensus among responders")
	}
	var dead Answer
	for _, a := range rep.Answers {
		if a.Resolver.Name == "Dead" {
			dead = a
		}
	}
	if dead.Err == "" || dead.Group != -1 {
		t.Errorf("dead resolver should carry an error and group -1, got err=%q group=%d", dead.Err, dead.Group)
	}
}

func TestCheckNormalizesOrderAndCase(t *testing.T) {
	resolvers := []Resolver{{Name: "A", IP: "1.1.1.1"}, {Name: "B", IP: "8.8.8.8"}}
	q := scripted(map[string]dnsx.Result{
		// same two MX values, different order and case / trailing dot
		"1.1.1.1": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("alt.MX.example.com."), rec("mx.example.com")}},
		"8.8.8.8": {Rcode: "NOERROR", Answers: []dnsx.Record{rec("mx.example.com"), rec("alt.mx.example.com")}},
	})
	rep := Check(context.Background(), q, "example.com", "MX", resolvers)
	if !rep.Consensus {
		t.Errorf("same values in different order/case should agree; groups=%d", len(rep.Groups))
	}
}
