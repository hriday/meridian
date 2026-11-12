// Package threatintel checks IP addresses against abuse.ch's Feodo Tracker
// botnet C2 blocklist — the command-and-control servers of malware families
// like Emotet, Dridex, QakBot and TrickBot. The feed is a free, keyless,
// downloadable JSON list (no account or API key), so a match means the IP has
// been seen acting as C2 infrastructure.
package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	// The full Feodo Tracker C2 blocklist (online + offline), keyless.
	feedURL   = "https://feodotracker.abuse.ch/downloads/ipblocklist.json"
	userAgent = "meridian (+https://github.com/hriday/meridian)"
)

// Doer is the slice of *http.Client used, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Hit is the verdict for one queried IP.
type Hit struct {
	IP         string `json:"ip"`
	Listed     bool   `json:"listed"` // seen as botnet C2 on the Feodo list
	Malware    string `json:"malware,omitempty"`
	Status     string `json:"status,omitempty"` // "online" / "offline" per abuse.ch
	Country    string `json:"country,omitempty"`
	ASNumber   int    `json:"asNumber,omitempty"`
	ASName     string `json:"asName,omitempty"`
	Port       int    `json:"port,omitempty"`
	FirstSeen  string `json:"firstSeen,omitempty"`
	LastOnline string `json:"lastOnline,omitempty"`
}

// feodoEntry mirrors one record in ipblocklist.json.
type feodoEntry struct {
	IPAddress  string `json:"ip_address"`
	Port       int    `json:"port"`
	Status     string `json:"status"`
	ASNumber   int    `json:"as_number"`
	ASName     string `json:"as_name"`
	Country    string `json:"country"`
	FirstSeen  string `json:"first_seen"`
	LastOnline string `json:"last_online"`
	Malware    string `json:"malware"`
}

// Check looks up each IP in the Feodo Tracker C2 blocklist and returns one Hit
// per distinct valid input IP, in the order first seen. Invalid entries are
// dropped; if nothing valid remains, no request is made.
func Check(ctx context.Context, client Doer, ips []string) ([]Hit, error) {
	// Deduplicate and validate first — so a page of junk never hits the network.
	var order []string
	seen := map[string]bool{}
	for _, raw := range ips {
		ip := strings.TrimSpace(raw)
		if ip == "" || net.ParseIP(ip) == nil || seen[ip] {
			continue
		}
		seen[ip] = true
		order = append(order, ip)
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("no valid IP addresses to check")
	}

	list, err := fetch(ctx, client)
	if err != nil {
		return nil, err
	}

	out := make([]Hit, 0, len(order))
	for _, ip := range order {
		h := Hit{IP: ip}
		if e, ok := list[ip]; ok {
			h.Listed = true
			h.Malware = e.Malware
			h.Status = e.Status
			h.Country = e.Country
			h.ASNumber = e.ASNumber
			h.ASName = e.ASName
			h.Port = e.Port
			h.FirstSeen = e.FirstSeen
			h.LastOnline = e.LastOnline
		}
		out = append(out, h)
	}
	return out, nil
}

func fetch(ctx context.Context, client Doer) (map[string]feodoEntry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach abuse.ch Feodo Tracker")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Feodo Tracker returned %s", resp.Status)
	}
	var entries []feodoEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("couldn't read the Feodo Tracker feed")
	}
	byIP := make(map[string]feodoEntry, len(entries))
	for _, e := range entries {
		// Keep the most recently seen record if an IP appears twice.
		if prev, ok := byIP[e.IPAddress]; !ok || e.LastOnline > prev.LastOnline {
			byIP[e.IPAddress] = e
		}
	}
	return byIP, nil
}

// ExtractIPs pulls the valid IPv4/IPv6 addresses out of a free-form blob
// (newlines, commas, spaces), preserving order and dropping anything that
// isn't an IP. Used to turn a pasted list into something to check.
func ExtractIPs(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	var out []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if net.ParseIP(f) != nil {
			out = append(out, f)
		}
	}
	return out
}
