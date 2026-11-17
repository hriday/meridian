package svcstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// fetch does a GET with the shared User-Agent and returns the body, mapping the
// common failure shapes to friendly errors.
func fetch(ctx context.Context, client Doer, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("couldn't reach the status page")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MiB cap
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("couldn't read the status feed")
	}
	return body, resp.StatusCode, nil
}

// checkStatuspage reads an Atlassian Statuspage /api/v2/summary.json feed.
func checkStatuspage(ctx context.Context, client Doer, p provider) (Status, error) {
	api := p.URL + "/api/v2/summary.json"
	body, code, err := fetch(ctx, client, api)
	if err != nil {
		return Status{}, err
	}
	if code == http.StatusNotFound {
		return Status{}, fmt.Errorf("no Statuspage feed at %s — is it a Statuspage-backed service?", p.URL)
	}
	if code != http.StatusOK {
		return Status{}, fmt.Errorf("the status page returned HTTP %d", code)
	}

	var raw summaryJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return Status{}, fmt.Errorf("couldn't read the status feed (not a Statuspage service?)")
	}

	name := raw.Page.Name
	if name == "" {
		name = p.Name
	}
	s := Status{
		Service:     name,
		URL:         p.URL,
		Source:      "Statuspage",
		Indicator:   raw.Status.Indicator,
		Description: raw.Status.Description,
		Operational: raw.Status.Indicator == "" || raw.Status.Indicator == "none",
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
