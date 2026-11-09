// Package trace runs an AS-aware traceroute: it finds the path to a host by
// sending ICMP Echo probes with an increasing TTL (unprivileged — no raw
// sockets), then labels each hop with its network (AS number + owner). The
// probing sits behind a Prober seam so the loop is testable without the
// network; this file holds the types and the pure aggregation helpers.
package trace

import (
	"fmt"
	"strings"
	"time"
)

// Hop is one TTL step of the path: the responding address (or "" when every
// probe timed out), its reverse-DNS host, the round-trip times of each probe,
// and the AS it belongs to once annotated.
type Hop struct {
	TTL     int             `json:"ttl"`
	Addr    string          `json:"addr"`
	Host    string          `json:"host"`
	RTTs    []time.Duration `json:"rtts"`
	ASN     string          `json:"asn"`
	ASOwner string          `json:"asOwner"`
	Timeout bool            `json:"timeout"`
}

// Result is a whole traceroute.
type Result struct {
	Target  string        `json:"target"`
	Addr    string        `json:"addr"`
	Hops    []Hop         `json:"hops"`
	Reached bool          `json:"reached"`
	Elapsed time.Duration `json:"elapsed"`
}

// rttStats returns the min, mean, and max of the non-timeout probe times
// (zero durations are treated as timeouts and ignored).
func rttStats(d []time.Duration) (min, avg, max time.Duration) {
	var sum time.Duration
	var n int
	for _, x := range d {
		if x <= 0 {
			continue
		}
		if n == 0 || x < min {
			min = x
		}
		if x > max {
			max = x
		}
		sum += x
		n++
	}
	if n == 0 {
		return 0, 0, 0
	}
	return min, sum / time.Duration(n), max
}

// asPath returns the distinct ASNs along the path, in first-seen order,
// carrying each one's owner (the networks the packets cross).
func asPath(hops []Hop) []Hop {
	seen := map[string]bool{}
	var out []Hop
	for _, h := range hops {
		if h.ASN == "" || seen[h.ASN] {
			continue
		}
		seen[h.ASN] = true
		out = append(out, Hop{ASN: h.ASN, ASOwner: h.ASOwner})
	}
	return out
}

// isBoundary reports whether hop i begins a new AS relative to the previous
// annotated hop (hops with no ASN are skipped when looking back).
func isBoundary(hops []Hop, i int) bool {
	if i < 0 || i >= len(hops) || hops[i].ASN == "" {
		return false
	}
	for j := i - 1; j >= 0; j-- {
		if hops[j].ASN != "" {
			return hops[j].ASN != hops[i].ASN
		}
	}
	return true // first annotated hop on the path
}

// Verdict is a plain-language, one-line summary of the trace.
func Verdict(r Result) string {
	if !r.Reached {
		return fmt.Sprintf("Did not reach %s — stopped after %d hops.", r.Target, len(r.Hops))
	}
	path := asPath(r.Hops)
	owners := make([]string, 0, len(path))
	for _, h := range path {
		if h.ASOwner != "" {
			owners = append(owners, h.ASOwner)
		} else {
			owners = append(owners, "AS"+h.ASN)
		}
	}
	target := r.Addr
	if r.Target != "" && r.Target != r.Addr {
		target = fmt.Sprintf("%s (%s)", r.Addr, r.Target)
	}
	word := "networks"
	if len(owners) == 1 {
		word = "network"
	}
	dest := "—"
	if len(r.Hops) > 0 {
		if _, avg, _ := rttStats(r.Hops[len(r.Hops)-1].RTTs); avg > 0 {
			dest = fmt.Sprintf("%.0f ms", float64(avg)/float64(time.Millisecond))
		}
	}
	return fmt.Sprintf("%d hops to %s, crossing %d %s — %s. Reached in %s.",
		len(r.Hops), target, len(owners), word, strings.Join(owners, " → "), dest)
}
