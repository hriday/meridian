package svcstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// googleIncident is one entry in Google's incidents.json (same shape for Google
// Cloud and Google Workspace). The feed includes resolved incidents as history;
// an incident is open iff it has no `end` timestamp and its most recent update
// isn't "AVAILABLE".
type googleIncident struct {
	ExternalDesc     string `json:"external_desc"`
	Begin            string `json:"begin"`
	End              string `json:"end"`
	StatusImpact     string `json:"status_impact"` // SERVICE_DISRUPTION | SERVICE_OUTAGE | SERVICE_INFORMATION
	Severity         string `json:"severity"`      // low | medium | high
	ServiceName      string `json:"service_name"`
	URI              string `json:"uri"`
	AffectedProducts []struct {
		Title string `json:"title"`
	} `json:"affected_products"`
	MostRecentUpdate struct {
		Status   string `json:"status"` // AVAILABLE | SERVICE_DISRUPTION | SERVICE_INFORMATION
		Modified string `json:"modified"`
	} `json:"most_recent_update"`
}

func checkGoogle(ctx context.Context, client Doer, p provider) (Status, error) {
	body, code, err := fetch(ctx, client, p.URL)
	if err != nil {
		return Status{}, err
	}
	if code != 200 {
		return Status{}, fmt.Errorf("Google's status feed returned HTTP %d", code)
	}

	var incidents []googleIncident
	if err := json.Unmarshal(body, &incidents); err != nil {
		return Status{}, fmt.Errorf("couldn't read Google's status feed")
	}

	base := strings.TrimSuffix(p.URL, "incidents.json")
	s := Status{Service: p.Name, URL: googleStatusPage(p), Source: "Google", Operational: true, Indicator: "none"}
	worst := "none"
	seen := map[string]bool{}
	for _, in := range incidents {
		if strings.TrimSpace(in.End) != "" || strings.EqualFold(in.MostRecentUpdate.Status, "AVAILABLE") {
			continue // resolved
		}
		s.Operational = false
		comp := googleComponentStatus(in.StatusImpact)
		impact := googleImpact(in.Severity)
		worst = worseImpact(worst, impact)
		for _, ap := range in.AffectedProducts {
			if n := strings.TrimSpace(ap.Title); n != "" && !seen[n] {
				seen[n] = true
				s.Impacted = append(s.Impacted, Component{Name: n, Status: comp})
			}
		}
		if len(in.AffectedProducts) == 0 && in.ServiceName != "" {
			s.Impacted = append(s.Impacted, Component{Name: in.ServiceName, Status: comp})
		}
		s.Incidents = append(s.Incidents, Incident{
			Name:      firstNonEmpty(in.ExternalDesc, in.ServiceName),
			Status:    "ongoing",
			Impact:    impact,
			Shortlink: base + strings.TrimPrefix(in.URI, "/"),
			UpdatedAt: in.MostRecentUpdate.Modified,
		})
	}
	if s.Operational {
		s.Description = "All services available"
	} else {
		s.Indicator = worst
		s.Description = fmt.Sprintf("%d ongoing incident(s)", len(s.Incidents))
	}
	return s, nil
}

func googleComponentStatus(statusImpact string) string {
	switch strings.ToUpper(strings.TrimSpace(statusImpact)) {
	case "SERVICE_OUTAGE":
		return "major_outage"
	case "SERVICE_DISRUPTION":
		return "partial_outage"
	case "SERVICE_INFORMATION":
		return "degraded_performance"
	default:
		return "partial_outage"
	}
}

func googleImpact(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "high":
		return "critical"
	case "medium":
		return "major"
	case "low":
		return "minor"
	default:
		return "minor"
	}
}

func googleStatusPage(p provider) string {
	if strings.Contains(p.URL, "appsstatus") {
		return "https://www.google.com/appsstatus/dashboard/"
	}
	return "https://status.cloud.google.com/"
}
