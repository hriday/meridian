// Package atlas schedules one-off ping measurements on the RIPE Atlas network
// and aggregates the per-probe results. The caller supplies their own Atlas API
// key; it is sent only to atlas.ripe.net in the Authorization header and is
// never logged or stored by this package.
package atlas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

// Doer is the subset of *http.Client the package needs; injected so tests stay
// offline.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// baseURL is the RIPE Atlas API root; a package var so tests can point it at an
// httptest server.
var baseURL = "https://atlas.ripe.net/api/v2/"

// userAgent carries no user identity.
const userAgent = "Meridian/1.0 (+network diagnostics)"

// Probe is one probe's ping result.
type Probe struct {
	ID       int     `json:"id"`
	From     string  `json:"from"`
	MinMs    float64 `json:"minMs"`
	AvgMs    float64 `json:"avgMs"`
	MaxMs    float64 `json:"maxMs"`
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
}

// Report aggregates a measurement across all responding probes.
type Report struct {
	Target        string  `json:"target"`
	MeasurementID int     `json:"measurementId"`
	Probes        []Probe `json:"probes"`
	Responding    int     `json:"responding"`
	MinMs         float64 `json:"minMs"`
	AvgMs         float64 `json:"avgMs"`
	MaxMs         float64 `json:"maxMs"`
	LossPct       float64 `json:"lossPct"`
}

// atlasError is the RIPE Atlas error envelope. The most specific message (e.g.
// the insufficient-credits text) often lives in the nested errors[] array while
// the top-level detail is a generic "There was a problem with your request".
type atlasError struct {
	Error struct {
		Detail string `json:"detail"`
		Status int    `json:"status"`
		Errors []struct {
			Detail string `json:"detail"`
		} `json:"errors"`
	} `json:"error"`
}

func afFor(target string) int {
	if addr, err := netip.ParseAddr(strings.TrimSpace(target)); err == nil && addr.Is6() {
		return 6
	}
	return 4
}

// schedule POSTs a one-off ping measurement and returns its id.
func schedule(ctx context.Context, d Doer, key, target string, probes int) (int, error) {
	body := map[string]any{
		"definitions": []map[string]any{{
			"type":        "ping",
			"af":          afFor(target),
			"target":      target,
			"description": "Meridian remote ping " + target,
			"is_oneoff":   true,
			"packets":     3,
		}},
		"probes": []map[string]any{{
			"requested": probes,
			"type":      "area",
			"value":     "WW",
		}},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"measurements/", bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Key "+key)
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, scheduleError(resp.StatusCode, data)
	}

	var out struct {
		Measurements []int `json:"measurements"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return 0, fmt.Errorf("atlas: unexpected response: %w", err)
	}
	if len(out.Measurements) == 0 {
		return 0, fmt.Errorf("atlas: no measurement id returned")
	}
	return out.Measurements[0], nil
}

// scheduleError maps an Atlas error body to a clear message, distinguishing the
// common credits and key cases.
func scheduleError(status int, body []byte) error {
	var ae atlasError
	_ = json.Unmarshal(body, &ae)

	// Collect every message in the envelope; the nested errors[] entries are the
	// most specific, so prefer the last of them for the fallback text.
	var parts []string
	if ae.Error.Detail != "" {
		parts = append(parts, ae.Error.Detail)
	}
	for _, e := range ae.Error.Errors {
		if e.Detail != "" {
			parts = append(parts, e.Detail)
		}
	}
	combined := strings.ToLower(strings.Join(parts, " "))

	switch {
	case strings.Contains(combined, "credit"):
		return fmt.Errorf("atlas: insufficient credits — host a probe or get Atlas credits to run measurements")
	case status == 401 || status == 403 ||
		strings.Contains(combined, "key") ||
		strings.Contains(combined, "auth") ||
		strings.Contains(combined, "permission"):
		return fmt.Errorf("atlas: the API key was rejected — check your RIPE Atlas key (it needs the \"Schedule a new measurement\" permission)")
	case len(parts) > 0:
		return fmt.Errorf("atlas: %s", parts[len(parts)-1])
	default:
		return fmt.Errorf("atlas: measurement request failed (HTTP %d)", status)
	}
}

// rawResult is one probe's ping result as RIPE Atlas reports it.
type rawResult struct {
	PrbID int     `json:"prb_id"`
	From  string  `json:"from"`
	Min   float64 `json:"min"`
	Avg   float64 `json:"avg"`
	Max   float64 `json:"max"`
	Sent  int     `json:"sent"`
	Rcvd  int     `json:"rcvd"`
}

// results fetches and parses the per-probe ping results for a measurement.
func results(ctx context.Context, d Doer, key string, id int) ([]Probe, error) {
	url := fmt.Sprintf("%smeasurements/%d/results/", baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Key "+key)
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, scheduleError(resp.StatusCode, data)
	}

	var raw []rawResult
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("atlas: unexpected results: %w", err)
	}
	probes := make([]Probe, 0, len(raw))
	for _, r := range raw {
		probes = append(probes, Probe{
			ID:       r.PrbID,
			From:     r.From,
			MinMs:    r.Min,
			AvgMs:    r.Avg,
			MaxMs:    r.Max,
			Sent:     r.Sent,
			Received: r.Rcvd,
		})
	}
	return probes, nil
}

// Ping schedules a one-off ping from the given number of worldwide probes, then
// polls results until some arrive or the timeout elapses, and aggregates them.
// A timeout with nothing reported yields an empty Report and no error.
func Ping(ctx context.Context, d Doer, key, target string, probes int, poll, timeout time.Duration) (Report, error) {
	id, err := schedule(ctx, d, key, target, probes)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Target: target, MeasurementID: id}

	deadline := time.Now().Add(timeout)
	for {
		ps, err := results(ctx, d, key, id)
		if err != nil {
			// A context deadline/cancel reached while polling is a timeout, not a
			// hard failure: return whatever reported so far (possibly nothing).
			if ctx.Err() != nil {
				return rep, nil
			}
			return Report{}, err
		}
		if len(ps) > 0 {
			rep.Probes = ps
			aggregate(&rep)
			return rep, nil
		}
		if !time.Now().Before(deadline) {
			return rep, nil // nothing reported in time
		}
		select {
		case <-ctx.Done():
			return rep, nil // timeout/cancel: graceful empty report
		case <-time.After(poll):
		}
	}
}

// aggregate fills the global min/avg/max/loss over responding probes.
func aggregate(rep *Report) {
	var sumAvg, sumLoss float64
	first := true
	for _, p := range rep.Probes {
		if p.Sent > 0 {
			sumLoss += float64(p.Sent-p.Received) / float64(p.Sent) * 100
		}
		if p.Received == 0 {
			continue
		}
		rep.Responding++
		sumAvg += p.AvgMs
		if first || p.MinMs < rep.MinMs {
			rep.MinMs = p.MinMs
		}
		if first || p.MaxMs > rep.MaxMs {
			rep.MaxMs = p.MaxMs
		}
		first = false
	}
	if rep.Responding > 0 {
		rep.AvgMs = sumAvg / float64(rep.Responding)
	}
	if len(rep.Probes) > 0 {
		rep.LossPct = sumLoss / float64(len(rep.Probes))
	}
}
