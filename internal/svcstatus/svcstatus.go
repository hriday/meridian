// Package svcstatus answers "is this online service having an outage right now?"
// from its public Statuspage.io feed (the /api/v2/summary.json a large share of
// major services publish) — free and keyless. It's the authoritative-status
// answer to a Downdetector-style question (Downdetector itself has no free API).
package svcstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Doer is the slice of *http.Client used, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

const userAgent = "meridian (+https://github.com/hriday/meridian)"

// Known maps a short slug to a service's Statuspage base URL. Custom URLs work
// too — any Statuspage-backed status page exposes /api/v2/summary.json.
var Known = map[string]string{
	"github":       "https://www.githubstatus.com",
	"cloudflare":   "https://www.cloudflarestatus.com",
	"discord":      "https://discordstatus.com",
	"reddit":       "https://www.redditstatus.com",
	"dropbox":      "https://status.dropbox.com",
	"twilio":       "https://status.twilio.com",
	"digitalocean": "https://status.digitalocean.com",
	"datadog":      "https://status.datadoghq.com",
	"atlassian":    "https://status.atlassian.com",
	"npm":          "https://status.npmjs.org",
	"pagerduty":    "https://status.pagerduty.com",
	"coinbase":     "https://status.coinbase.com",
	"openai":       "https://status.openai.com",
	"zoom":         "https://status.zoom.us",
}

// KnownSlugs returns the known service slugs, sorted.
func KnownSlugs() []string {
	out := make([]string, 0, len(Known))
	for k := range Known {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type Component struct {
	Name   string `json:"name"`
	Status string `json:"status"` // operational | degraded_performance | partial_outage | major_outage | under_maintenance
}

type Incident struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // investigating | identified | monitoring
	Impact    string `json:"impact"` // none | minor | major | critical
	Shortlink string `json:"shortlink"`
	UpdatedAt string `json:"updatedAt"`
}

// Status is the distilled result for one service.
type Status struct {
	Service     string      `json:"service"`
	URL         string      `json:"url"`
	Indicator   string      `json:"indicator"`   // none | minor | major | critical
	Description string      `json:"description"` // e.g. "All Systems Operational"
	Operational bool        `json:"operational"` // indicator == none
	Impacted    []Component `json:"impacted"`    // components not fully operational
	Incidents   []Incident  `json:"incidents"`   // unresolved incidents
}

// Check fetches and distills a service's status. target is a known slug, a
// status-page host, or a full URL.
func Check(ctx context.Context, client Doer, target string) (Status, error) {
	base, err := resolveBase(target)
	if err != nil {
		return Status{}, err
	}
	api := base + "/api/v2/summary.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, nil)
	if err != nil {
		return Status{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return Status{}, fmt.Errorf("couldn't reach the status page")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Status{}, fmt.Errorf("no Statuspage feed at %s — is it a Statuspage-backed service?", base)
	}
	if resp.StatusCode != http.StatusOK {
		return Status{}, fmt.Errorf("the status page returned %s", resp.Status)
	}

	var raw summaryJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Status{}, fmt.Errorf("couldn't read the status feed (not a Statuspage service?)")
	}

	s := Status{
		Service:     raw.Page.Name,
		URL:         base,
		Indicator:   raw.Status.Indicator,
		Description: raw.Status.Description,
		Operational: raw.Status.Indicator == "" || raw.Status.Indicator == "none",
	}
	if s.Service == "" {
		s.Service = base
	}
	for _, c := range raw.Components {
		if c.Status != "" && c.Status != "operational" {
			s.Impacted = append(s.Impacted, Component{Name: c.Name, Status: c.Status})
		}
	}
	for _, in := range raw.Incidents {
		if in.Status == "resolved" || in.Status == "postmortem" {
			continue
		}
		s.Incidents = append(s.Incidents, Incident{
			Name: in.Name, Status: in.Status, Impact: in.Impact,
			Shortlink: in.Shortlink, UpdatedAt: in.UpdatedAt,
		})
	}
	return s, nil
}

type summaryJSON struct {
	Page struct {
		Name string `json:"name"`
	} `json:"page"`
	Status struct {
		Indicator   string `json:"indicator"`
		Description string `json:"description"`
	} `json:"status"`
	Components []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"components"`
	Incidents []struct {
		Name      string `json:"name"`
		Status    string `json:"status"`
		Impact    string `json:"impact"`
		Shortlink string `json:"shortlink"`
		UpdatedAt string `json:"updated_at"`
	} `json:"incidents"`
}

// resolveBase turns a slug / host / URL into a scheme://host base.
func resolveBase(target string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(target))
	if t == "" {
		return "", fmt.Errorf("enter a service name or status-page URL")
	}
	if b, ok := Known[t]; ok {
		return b, nil
	}
	if strings.Contains(t, "://") {
		u, err := url.Parse(t)
		if err != nil || u.Host == "" {
			return "", fmt.Errorf("that doesn't look like a URL")
		}
		return u.Scheme + "://" + u.Host, nil
	}
	if strings.Contains(t, ".") { // bare host
		return "https://" + strings.TrimSuffix(t, "/"), nil
	}
	return "", fmt.Errorf("unknown service %q — paste its status-page URL, or try one of: %s", target, strings.Join(KnownSlugs(), ", "))
}
