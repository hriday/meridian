// Package xon checks an email address against XposedOrNot's free, keyless breach
// API (https://xposedornot.com/api_doc). Unlike the Pwned Passwords check, there
// is no k-anonymity for an email lookup: the address itself is sent to the
// service. Callers must make that clear to the user. User-initiated only.
package xon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	checkEmailURL = "https://api.xposedornot.com/v1/check-email/"
	userAgent     = "meridian (+https://github.com/hriday/meridian)"
)

// Doer is the slice of *http.Client used, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// CheckEmail returns the names of the breaches an email address appears in,
// according to XposedOrNot. An address in no known breach returns an empty slice
// (not an error). The address is sent to the service.
func CheckEmail(ctx context.Context, client Doer, email string) ([]string, error) {
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("enter an email address")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkEmailURL+url.PathEscape(email), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach the breach service")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNotFound:
		// 200 with results, or a body that signals "not found".
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("the free breach service is rate-limited right now — try again in a bit")
	default:
		return nil, fmt.Errorf("the breach service returned %s", resp.Status)
	}

	var body struct {
		Breaches [][]string `json:"breaches"`
		Error    string     `json:"Error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("couldn't read the breach result")
	}
	if body.Error != "" || len(body.Breaches) == 0 {
		return []string{}, nil // not found in any known breach
	}
	// De-duplicate and drop blanks, preserving order.
	seen := map[string]bool{}
	var names []string
	for _, n := range body.Breaches[0] {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names, nil
}
