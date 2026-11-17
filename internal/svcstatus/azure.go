package svcstatus

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

const azureFeed = "https://azure.status.microsoft/en-us/status/feed/"

// azureRSS is Azure's status RSS. When Azure is healthy the channel has zero
// <item>s (verified against the live feed); each item is an active advisory.
type azureRSS struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Description string `xml:"description"`
			Link        string `xml:"link"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

func checkAzure(ctx context.Context, client Doer, p provider) (Status, error) {
	body, code, err := fetch(ctx, client, p.URL)
	if err != nil {
		return Status{}, err
	}
	if code != 200 {
		return Status{}, fmt.Errorf("Azure's status feed returned HTTP %d", code)
	}

	var feed azureRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		return Status{}, fmt.Errorf("couldn't read Azure's status feed")
	}

	s := Status{
		Service:     p.Name,
		URL:         "https://azure.status.microsoft/en-us/status/",
		Source:      "Azure",
		Operational: len(feed.Channel.Items) == 0,
		Indicator:   "none",
	}
	if s.Operational {
		s.Description = "No active Azure service issues"
		return s, nil
	}
	// Azure's feed carries no severity, so we report presence honestly as
	// "minor" (active advisory) and let the titles speak.
	s.Indicator = "minor"
	s.Description = fmt.Sprintf("%d active Azure service issue(s)", len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		title := strings.TrimSpace(it.Title)
		s.Impacted = append(s.Impacted, Component{Name: title, Status: "partial_outage"})
		s.Incidents = append(s.Incidents, Incident{
			Name:      title,
			Status:    "active",
			Impact:    "minor",
			Shortlink: strings.TrimSpace(it.Link),
			UpdatedAt: strings.TrimSpace(it.PubDate),
		})
	}
	return s, nil
}
