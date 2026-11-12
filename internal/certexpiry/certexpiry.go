// Package certexpiry checks how soon TLS certificates expire across a list of
// hosts. It's the bulk companion to the single-cert TLS inspector: paste a
// fleet of host:port endpoints and get back days-to-expiry, soonest first, so
// the cert about to lapse is at the top. Everything is a direct TLS dial — no
// third-party service is involved.
package certexpiry

import (
	"context"
	"math"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Leaf is the part of a server's leaf certificate this tool reports on.
type Leaf struct {
	NotBefore time.Time
	NotAfter  time.Time
	Subject   string
	Issuer    string
	DNSNames  []string
}

// Probe fetches the leaf certificate presented by hostport (already normalized
// to host:port). The real implementation dials TLS; tests inject a fake.
type Probe func(ctx context.Context, hostport string) (Leaf, error)

// Result is one host's verdict.
type Result struct {
	Host     string `json:"host"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Subject  string `json:"subject,omitempty"`
	Issuer   string `json:"issuer,omitempty"`
	NotAfter string `json:"notAfter,omitempty"` // YYYY-MM-DD
	DaysLeft int    `json:"daysLeft"`           // negative once expired
	Expired  bool   `json:"expired"`
	NotYet   bool   `json:"notYet"` // not valid yet (NotBefore in the future)
}

// Check probes each host for its leaf certificate and returns one Result per
// distinct host, sorted so the most urgent (already expired, then soonest to
// expire) comes first; hosts that couldn't be reached sort to the end. now is
// injected so expiry math is deterministic.
func Check(ctx context.Context, probe Probe, hosts []string, now time.Time) []Result {
	var order []string
	seen := map[string]bool{}
	for _, h := range hosts {
		hp := NormalizeHost(h)
		if hp == "" || seen[hp] {
			continue
		}
		seen[hp] = true
		order = append(order, hp)
	}
	if len(order) == 0 {
		return nil
	}

	results := make([]Result, 0, len(order))
	for _, hp := range order {
		r := Result{Host: hp}
		leaf, err := probe(ctx, hp)
		if err != nil {
			r.Error = err.Error()
			results = append(results, r)
			continue
		}
		r.OK = true
		r.Subject = leaf.Subject
		r.Issuer = leaf.Issuer
		r.NotAfter = leaf.NotAfter.Format("2006-01-02")
		r.DaysLeft = daysBetween(now, leaf.NotAfter)
		r.Expired = leaf.NotAfter.Before(now)
		r.NotYet = !leaf.NotBefore.IsZero() && leaf.NotBefore.After(now)
		results = append(results, r)
	}

	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.OK != b.OK {
			return a.OK // successes before errors
		}
		if !a.OK {
			return false // keep errors in input order
		}
		return a.DaysLeft < b.DaysLeft // soonest-to-expire first
	})
	return results
}

// daysBetween returns whole days from now until t, floored toward negative
// infinity: "10.9 days left" reads as 10, and a cert that lapsed 11.2 days ago
// reads as -12 (how people say "it's been 12 days").
func daysBetween(now, t time.Time) int {
	return int(math.Floor(t.Sub(now).Hours() / 24))
}

// NormalizeHost turns a user-entered host into host:port, defaulting to :443 and
// stripping any URL scheme/path. Returns "" for blank input.
func NormalizeHost(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Strip a scheme + path if someone pasted a URL.
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Host
		}
	} else if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	if _, _, err := net.SplitHostPort(s); err == nil {
		return s // already host:port
	}
	return net.JoinHostPort(s, "443")
}

// SplitHosts breaks a free-form blob (newlines, commas, spaces) into raw host
// tokens, preserving order and dropping empties. Normalization happens later.
func SplitHosts(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}
