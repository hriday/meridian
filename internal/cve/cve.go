// Package cve looks up vulnerabilities from the NVD (National Vulnerability
// Database) API 2.0 — free and keyless (rate-limited to a few requests per 30s
// without a key, which is fine for interactive lookups). It resolves a CVE by id
// or searches by keyword, and flags CVEs CISA lists as known-exploited.
package cve

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

const (
	apiBase   = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	userAgent = "meridian (+https://github.com/hriday/meridian)"
)

// Doer is the slice of *http.Client used, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// CVE is one vulnerability, flattened from the NVD record.
type CVE struct {
	ID          string   `json:"id"`
	Published   string   `json:"published"`
	Description string   `json:"description"`
	CVSS        float64  `json:"cvss"`     // base score, 0 if none
	Severity    string   `json:"severity"` // CRITICAL/HIGH/MEDIUM/LOW
	Vector      string   `json:"vector"`
	CWE         []string `json:"cwe"`
	KEV         bool     `json:"kev"` // CISA Known Exploited Vulnerability
	References  []string `json:"references"`
}

var cveID = regexp.MustCompile(`(?i)^CVE-\d{4}-\d{4,}$`)

// IsID reports whether s looks like a CVE identifier.
func IsID(s string) bool { return cveID.MatchString(strings.TrimSpace(s)) }

// Lookup resolves a single CVE by its id (e.g. CVE-2021-44228).
func Lookup(ctx context.Context, client Doer, id string) (CVE, error) {
	id = strings.ToUpper(strings.TrimSpace(id))
	if !IsID(id) {
		return CVE{}, fmt.Errorf("that's not a CVE id (expected CVE-YYYY-NNNN)")
	}
	cves, err := fetch(ctx, client, url.Values{"cveId": {id}})
	if err != nil {
		return CVE{}, err
	}
	if len(cves) == 0 {
		return CVE{}, fmt.Errorf("no record found for %s", id)
	}
	return cves[0], nil
}

// Search returns CVEs matching a keyword (most recent first-ish, capped by NVD).
func Search(ctx context.Context, client Doer, keyword string) ([]CVE, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("enter a CVE id or a keyword (e.g. a product name)")
	}
	return fetch(ctx, client, url.Values{"keywordSearch": {keyword}, "resultsPerPage": {"30"}})
}

func fetch(ctx context.Context, client Doer, q url.Values) ([]CVE, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach the NVD")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("the NVD is rate-limiting (it's keyless and throttled) — wait ~30s and retry")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the NVD returned %s", resp.Status)
	}
	var body nvdResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("couldn't read the NVD response")
	}
	out := make([]CVE, 0, len(body.Vulnerabilities))
	for _, v := range body.Vulnerabilities {
		out = append(out, v.CVE.flatten())
	}
	return out, nil
}

type nvdResp struct {
	Vulnerabilities []struct {
		CVE nvdCVE `json:"cve"`
	} `json:"vulnerabilities"`
}

type nvdCVE struct {
	ID             string `json:"id"`
	Published      string `json:"published"`
	CisaExploitAdd string `json:"cisaExploitAdd"`
	Descriptions   []struct {
		Lang  string `json:"lang"`
		Value string `json:"value"`
	} `json:"descriptions"`
	Metrics struct {
		V31 []nvdMetric `json:"cvssMetricV31"`
		V30 []nvdMetric `json:"cvssMetricV30"`
	} `json:"metrics"`
	Weaknesses []struct {
		Description []struct {
			Value string `json:"value"`
		} `json:"description"`
	} `json:"weaknesses"`
	References []struct {
		URL string `json:"url"`
	} `json:"references"`
}

type nvdMetric struct {
	CvssData struct {
		BaseScore    float64 `json:"baseScore"`
		BaseSeverity string  `json:"baseSeverity"`
		VectorString string  `json:"vectorString"`
	} `json:"cvssData"`
}

func (c nvdCVE) flatten() CVE {
	out := CVE{ID: c.ID, Published: date(c.Published), KEV: strings.TrimSpace(c.CisaExploitAdd) != ""}
	for _, d := range c.Descriptions {
		if d.Lang == "en" {
			out.Description = strings.TrimSpace(d.Value)
			break
		}
	}
	m := c.Metrics.V31
	if len(m) == 0 {
		m = c.Metrics.V30
	}
	if len(m) > 0 {
		out.CVSS = m[0].CvssData.BaseScore
		out.Severity = m[0].CvssData.BaseSeverity
		out.Vector = m[0].CvssData.VectorString
	}
	seenCWE := map[string]bool{}
	for _, w := range c.Weaknesses {
		for _, d := range w.Description {
			if v := strings.TrimSpace(d.Value); strings.HasPrefix(v, "CWE-") && !seenCWE[v] {
				seenCWE[v] = true
				out.CWE = append(out.CWE, v)
			}
		}
	}
	seenRef := map[string]bool{}
	for _, r := range c.References {
		if r.URL != "" && !seenRef[r.URL] {
			seenRef[r.URL] = true
			out.References = append(out.References, r.URL)
		}
		if len(out.References) >= 12 {
			break
		}
	}
	return out
}

func date(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
