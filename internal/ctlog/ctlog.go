// Package ctlog discovers the subdomains of a domain from public Certificate
// Transparency logs via crt.sh. CT logs record every TLS certificate issued, so
// this surfaces hosts that have ever had a cert — a fast, free, keyless way to
// map a domain's attack surface. It finds only names that appear in a cert, not
// every host that exists. User-initiated; nothing runs automatically.
package ctlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
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
}

type entry struct {
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"` // newline-separated SANs
	NotBefore  string `json:"not_before"`
	NotAfter   string `json:"not_after"`
}

// Subdomains returns the domain's subdomains found in CT logs (via crt.sh),
// sorted by name. The apex domain itself is included if it appears.
func Subdomains(ctx context.Context, client Doer, domain string) ([]Subdomain, error) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return nil, fmt.Errorf("enter a domain, e.g. example.com")
	}
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
		return nil, fmt.Errorf("crt.sh returned %s (it can be flaky — try again)", resp.Status)
	}

	var entries []entry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("couldn't read the CT log results")
	}
	return aggregate(entries, domain), nil
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
