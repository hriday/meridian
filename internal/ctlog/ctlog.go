// Package ctlog discovers the subdomains of a domain from public Certificate
// Transparency logs — crt.sh first, with a certSpotter fallback if crt.sh is
// unresponsive. CT logs record every TLS certificate issued, so this surfaces
// hosts that have ever had a cert — a fast, free, keyless way to map a domain's
// attack surface. It finds only names that appear in a cert, not every host that
// exists. User-initiated; nothing runs automatically.
package ctlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const userAgent = "meridian (+https://github.com/hriday/meridian)"

// Doer is the slice of *http.Client used, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Subdomain is one hostname discovered in CT logs, aggregated across its certs.
type Subdomain struct {
	Name      string `json:"name"`
	Wildcard  bool   `json:"wildcard"`  // appeared as *.name
	Certs     int    `json:"certs"`     // number of CT entries mentioning it
	FirstSeen string `json:"firstSeen"` // earliest not_before (date)
	LastSeen  string `json:"lastSeen"`  // latest not_after (date)
	Stale     bool   `json:"stale"`     // newest cert expired well over a year ago — likely a retired host
}

type entry struct {
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"` // newline-separated SANs
	NotBefore  string `json:"not_before"`
	NotAfter   string `json:"not_after"`
}

// Subdomains returns the domain's subdomains found in CT logs, sorted by name.
// It queries crt.sh first and, if crt.sh is unreachable or errors (it's often
// flaky), falls back to the certSpotter API so a single slow source doesn't
// break the lookup. The apex domain itself is included if it appears.
func Subdomains(ctx context.Context, client Doer, domain string) ([]Subdomain, error) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return nil, fmt.Errorf("enter a domain, e.g. example.com")
	}
	// Bound the crt.sh attempt to half the budget, so a *slow* (not just down)
	// crt.sh still leaves time for the backup.
	crtCtx := ctx
	if dl, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		crtCtx, cancel = context.WithTimeout(ctx, time.Until(dl)/2)
		defer cancel()
	}
	entries, err := fetchCrtSh(crtCtx, client, domain)
	if err != nil {
		// crt.sh down/flaky/slow → try the backup with the remaining budget.
		cs, csErr := fetchCertSpotter(ctx, client, domain)
		if csErr != nil {
			return nil, fmt.Errorf("CT lookup failed (crt.sh: %v; certSpotter: %v)", err, csErr)
		}
		entries = cs
	}
	out := aggregate(entries, domain)
	markStale(out, time.Now())
	return out, nil
}

// staleAfter is how long past a subdomain's newest certificate expiry we flag it
// as likely retired: two full ~398-day cert lifetimes with no renewal in CT.
const staleAfter = 2 * 365 * 24 * time.Hour

// markStale flags subdomains whose most recent certificate (LastSeen = latest
// not_after) expired more than staleAfter ago — a strong hint the host is
// decommissioned and its DNS record can be pulled. It is only a hint: the record
// may still resolve, so callers pair it with a live lookup before deleting.
func markStale(subs []Subdomain, now time.Time) {
	for i := range subs {
		t, err := time.Parse("2006-01-02", subs[i].LastSeen)
		if err != nil {
			continue
		}
		if now.Sub(t) > staleAfter {
			subs[i].Stale = true
		}
	}
}

// fetchCrtSh queries crt.sh for every cert mentioning *.domain.
func fetchCrtSh(ctx context.Context, client Doer, domain string) ([]entry, error) {
	u := url.URL{Scheme: "https", Host: "crt.sh", Path: "/"}
	q := u.Query()
	q.Set("q", "%."+domain) // % wildcard → every sub.domain
	q.Set("output", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach crt.sh")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned %s", resp.Status)
	}
	var entries []entry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("couldn't read the crt.sh results")
	}
	return entries, nil
}

// fetchCertSpotter queries the certSpotter API (free, keyless) and maps its
// issuances into the same entry shape aggregate expects.
func fetchCertSpotter(ctx context.Context, client Doer, domain string) ([]entry, error) {
	u := url.URL{Scheme: "https", Host: "api.certspotter.com", Path: "/v1/issuances"}
	q := u.Query()
	q.Set("domain", domain)
	q.Set("include_subdomains", "true")
	q.Set("expand", "dns_names")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach certSpotter")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("certSpotter returned %s", resp.Status)
	}
	var items []struct {
		DNSNames  []string `json:"dns_names"`
		NotBefore string   `json:"not_before"`
		NotAfter  string   `json:"not_after"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("couldn't read the certSpotter results")
	}
	entries := make([]entry, 0, len(items))
	for _, it := range items {
		entries = append(entries, entry{
			NameValue: strings.Join(it.DNSNames, "\n"),
			NotBefore: it.NotBefore,
			NotAfter:  it.NotAfter,
		})
	}
	return entries, nil
}

// aggregate folds raw cert entries into one record per hostname. A name is
// counted once per CERT entry, even when it appears in both the CN and the SANs.
func aggregate(entries []entry, domain string) []Subdomain {
	byName := map[string]*Subdomain{}
	for _, e := range entries {
		// Distinct in-scope names in this cert (name -> wildcard).
		names := map[string]bool{}
		add := func(raw string) {
			name := strings.ToLower(strings.TrimSpace(raw))
			wildcard := strings.HasPrefix(name, "*.")
			name = strings.TrimPrefix(name, "*.")
			if name == "" || !inScope(name, domain) {
				return
			}
			names[name] = names[name] || wildcard
		}
		add(e.CommonName)
		for _, n := range strings.Split(e.NameValue, "\n") {
			add(n)
		}
		for name, wildcard := range names {
			s := byName[name]
			if s == nil {
				s = &Subdomain{Name: name}
				byName[name] = s
			}
			s.Certs++
			if wildcard {
				s.Wildcard = true
			}
			if d := date(e.NotBefore); d != "" && (s.FirstSeen == "" || d < s.FirstSeen) {
				s.FirstSeen = d
			}
			if d := date(e.NotAfter); d != "" && d > s.LastSeen {
				s.LastSeen = d
			}
		}
	}
	out := make([]Subdomain, 0, len(byName))
	for _, s := range byName {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func inScope(name, domain string) bool {
	return name == domain || strings.HasSuffix(name, "."+domain)
}

func date(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// normalizeDomain strips a scheme/path/port and a leading wildcard or www, lowercasing.
func normalizeDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	if i := strings.Index(d, "://"); i >= 0 {
		d = d[i+3:]
	}
	d = strings.TrimPrefix(d, "*.")
	if i := strings.IndexAny(d, "/:"); i >= 0 {
		d = d[:i]
	}
	d = strings.Trim(d, ".")
	if !strings.Contains(d, ".") || strings.ContainsAny(d, " \t") {
		return ""
	}
	return d
}
