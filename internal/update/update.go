// Package update checks whether a newer Meridian release is available on GitHub.
// It makes a single, user-initiated request to the public GitHub Releases API,
// carries a generic User-Agent, and sends no identifying information — there is
// no telemetry and no automatic background polling.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	releasesAPI = "https://api.github.com/repos/hriday/meridian/releases/latest"
	userAgent   = "meridian/0.1 (+https://github.com/hriday/meridian)"
)

// Result is the outcome of a version check.
type Result struct {
	Current string `json:"current"` // the running build's version
	Latest  string `json:"latest"`  // the latest released tag, or "" on failure
	Newer   bool   `json:"newer"`   // a newer release is available
	URL     string `json:"url"`     // the release page
	Notes   string `json:"notes"`   // release notes (truncated)
}

var client = &http.Client{Timeout: 12 * time.Second}

// CheckLatest asks the public GitHub Releases API for the latest Meridian
// release and compares it to current.
func CheckLatest(ctx context.Context, current string) (Result, error) {
	return checkLatest(ctx, current, releasesAPI)
}

func checkLatest(ctx context.Context, current, url string) (Result, error) {
	res := Result{Current: current}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return res, fmt.Errorf("couldn't reach GitHub to check for updates")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return res, fmt.Errorf("no published releases found")
	}
	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("GitHub returned %s", resp.Status)
	}

	var gr struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return res, fmt.Errorf("couldn't read the release info")
	}

	res.Latest = strings.TrimSpace(gr.TagName)
	res.URL = gr.HTMLURL
	res.Notes = truncate(strings.TrimSpace(gr.Body), 600)
	res.Newer = isNewer(current, res.Latest)
	return res, nil
}

// isNewer reports whether latest is a newer semantic version than current. A
// current that doesn't parse as a version (e.g. "dev") is treated as older than
// any real release; an empty latest is never newer.
func isNewer(current, latest string) bool {
	if strings.TrimSpace(latest) == "" {
		return false
	}
	c, l := parseVer(current), parseVer(latest)
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// parseVer extracts [major, minor, patch] from a tag like "v1.2.3", ignoring a
// leading "v" and any pre-release/build suffix. Unparseable parts are 0.
func parseVer(s string) [3]int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	var out [3]int
	for i, p := range strings.SplitN(s, ".", 3) {
		if i > 2 {
			break
		}
		out[i], _ = strconv.Atoi(strings.TrimSpace(p))
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}
