package svcstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	appleConsumerFeed  = "https://www.apple.com/support/systemstatus/data/system_status_en_US.js"
	appleDeveloperFeed = "https://www.apple.com/support/systemstatus/data/developer/system_status_en_US.js"
)

// appleFeed is Apple's system-status data: a list of services, each with an
// events array. The consumer feed is raw JSON; the developer feed is JSONP
// (jsonCallback({...});). A service is down now iff it has an event whose
// eventStatus is "ongoing" (resolved events stay in the list as history).
type appleFeed struct {
	Services []struct {
		ServiceName string `json:"serviceName"`
		Events      []struct {
			StatusType    string `json:"statusType"`  // Outage | Issue | Performance
			EventStatus   string `json:"eventStatus"` // ongoing | resolved
			Message       string `json:"message"`
			UsersAffected string `json:"usersAffected"`
			DatePosted    string `json:"datePosted"`
			StartDate     string `json:"startDate"`
		} `json:"events"`
	} `json:"services"`
}

func checkApple(ctx context.Context, client Doer, p provider) (Status, error) {
	body, code, err := fetch(ctx, client, p.URL)
	if err != nil {
		return Status{}, err
	}
	if code != 200 {
		return Status{}, fmt.Errorf("Apple's status feed returned HTTP %d", code)
	}

	var feed appleFeed
	if err := json.Unmarshal(stripJSONP(body), &feed); err != nil {
		return Status{}, fmt.Errorf("couldn't read Apple's status feed")
	}

	s := Status{Service: p.Name, URL: appleStatusPage(p), Source: "Apple", Operational: true, Indicator: "none"}
	worst := "none"
	for _, svc := range feed.Services {
		name := strings.TrimSpace(svc.ServiceName)
		for _, ev := range svc.Events {
			if !strings.EqualFold(ev.EventStatus, "ongoing") {
				continue
			}
			s.Operational = false
			comp, impact := appleSeverity(ev.StatusType)
			worst = worseImpact(worst, impact)
			s.Impacted = append(s.Impacted, Component{Name: name, Status: comp})
			title := name
			if ev.UsersAffected != "" {
				title += " — " + ev.UsersAffected
			}
			s.Incidents = append(s.Incidents, Incident{
				Name: title, Status: "ongoing", Impact: impact, UpdatedAt: firstNonEmpty(ev.DatePosted, ev.StartDate),
			})
			break // one ongoing event per service is enough to mark it down
		}
	}
	if s.Operational {
		s.Description = "All services operational"
	} else {
		s.Indicator = worst
		s.Description = fmt.Sprintf("%d service(s) with an ongoing issue", len(s.Impacted))
	}
	return s, nil
}

func appleSeverity(statusType string) (component, impact string) {
	switch strings.ToLower(strings.TrimSpace(statusType)) {
	case "outage":
		return "major_outage", "major"
	case "issue":
		return "partial_outage", "minor"
	case "performance":
		return "degraded_performance", "minor"
	default:
		return "partial_outage", "minor"
	}
}

func appleStatusPage(p provider) string {
	if p.URL == appleDeveloperFeed {
		return "https://developer.apple.com/system-status/"
	}
	return "https://www.apple.com/support/systemstatus/"
}

// stripJSONP unwraps a JSONP body like `jsonCallback({...});` to the bare JSON.
// A body that's already JSON (starts with { or [) is returned unchanged.
func stripJSONP(body []byte) []byte {
	s := strings.TrimSpace(string(body))
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return []byte(s)
	}
	open := strings.IndexAny(s, "{[")
	close := strings.LastIndexAny(s, "}]")
	if open >= 0 && close > open {
		return []byte(s[open : close+1])
	}
	return body
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// worseImpact returns the more severe of two impact levels.
func worseImpact(a, b string) string {
	rank := map[string]int{"none": 0, "minor": 1, "major": 2, "critical": 3}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
