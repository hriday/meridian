// Package secheaders grades a URL's live HTTP security response headers and maps
// each finding to its OWASP Top 10 (2021) category. It reads the headers of the
// running server — distinct from the config analyzers, which read a config file.
// Free, keyless, user-initiated.
package secheaders

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const userAgent = "meridian (+https://github.com/hriday/meridian)"

// Grade values for a single header.
const (
	GradeOK      = "ok"
	GradeWarn    = "warn"
	GradeMissing = "missing"
	GradeInfo    = "info" // information disclosure (header that should be absent)
)

// Header is one checked security header's result.
type Header struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Value   string `json:"value"`
	Grade   string `json:"grade"`
	Note    string `json:"note"`
	OWASP   string `json:"owasp"` // e.g. "A05 Security Misconfiguration"
}

// Report is the overall result for a URL.
type Report struct {
	URL      string   `json:"url"`
	FinalURL string   `json:"finalUrl"`
	HTTPS    bool     `json:"https"`
	Status   int      `json:"status"`
	Headers  []Header `json:"headers"`
	Score    int      `json:"score"` // 0–100
	Grade    string   `json:"grade"` // A–F
}

// Scan fetches rawURL (following redirects) and grades its security headers.
func Scan(ctx context.Context, client *http.Client, rawURL string) (Report, error) {
	rawURL = normalizeURL(rawURL)
	if rawURL == "" {
		return Report{}, fmt.Errorf("enter a URL, e.g. example.com")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Report{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("couldn't reach %s", rawURL)
	}
	defer resp.Body.Close()
	final := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		final = resp.Request.URL.String()
	}
	rep := classify(resp.Header, final, resp.StatusCode)
	rep.URL = rawURL
	return rep, nil
}

func normalizeURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return u
}

// classify grades the headers and maps each to an OWASP Top 10 (2021) category.
func classify(h http.Header, finalURL string, status int) Report {
	https := strings.HasPrefix(strings.ToLower(finalURL), "https://")
	rep := Report{FinalURL: finalURL, HTTPS: https, Status: status}

	get := func(name string) string { return strings.TrimSpace(h.Get(name)) }
	csp := get("Content-Security-Policy")

	// Strict-Transport-Security (HSTS).
	hsts := get("Strict-Transport-Security")
	switch {
	case !https:
		rep.add("Strict-Transport-Security", false, "", GradeWarn, "A02 Cryptographic Failures",
			"Not served over HTTPS — HSTS can't apply; serve the site over TLS.")
	case hsts == "":
		rep.add("Strict-Transport-Security", false, "", GradeMissing, "A02 Cryptographic Failures",
			"No HSTS — browsers may still try plain HTTP, allowing downgrade/MITM. Add it with a long max-age.")
	case maxAge(hsts) < 15552000:
		rep.add("Strict-Transport-Security", true, hsts, GradeWarn, "A02 Cryptographic Failures",
			"HSTS present but max-age is short (<180 days). Use at least 15552000 and includeSubDomains.")
	default:
		rep.add("Strict-Transport-Security", true, hsts, GradeOK, "A02 Cryptographic Failures",
			"HSTS enforced — browsers stick to HTTPS.")
	}

	// Content-Security-Policy.
	switch {
	case csp == "":
		rep.add("Content-Security-Policy", false, "", GradeMissing, "A03 Injection (XSS)",
			"No CSP — the strongest defence against cross-site scripting is absent.")
	case strings.Contains(strings.ToLower(csp), "unsafe-inline") || strings.Contains(strings.ToLower(csp), "unsafe-eval"):
		rep.add("Content-Security-Policy", true, csp, GradeWarn, "A03 Injection (XSS)",
			"CSP present but allows unsafe-inline/unsafe-eval, which weakens XSS protection.")
	default:
		rep.add("Content-Security-Policy", true, csp, GradeOK, "A03 Injection (XSS)",
			"CSP set — restricts where scripts and resources may load from.")
	}

	// Clickjacking: X-Frame-Options or CSP frame-ancestors.
	xfo := get("X-Frame-Options")
	frameAnc := strings.Contains(strings.ToLower(csp), "frame-ancestors")
	switch {
	case xfo == "" && !frameAnc:
		rep.add("X-Frame-Options", false, "", GradeMissing, "A05 Security Misconfiguration",
			"No X-Frame-Options or CSP frame-ancestors — the page can be framed (clickjacking).")
	case xfo == "" && frameAnc:
		rep.add("X-Frame-Options", true, "(CSP frame-ancestors)", GradeOK, "A05 Security Misconfiguration",
			"Framing controlled via CSP frame-ancestors.")
	default:
		rep.add("X-Frame-Options", true, xfo, GradeOK, "A05 Security Misconfiguration",
			"Framing restricted — protects against clickjacking.")
	}

	// X-Content-Type-Options.
	xcto := get("X-Content-Type-Options")
	if strings.EqualFold(xcto, "nosniff") {
		rep.add("X-Content-Type-Options", true, xcto, GradeOK, "A05 Security Misconfiguration",
			"MIME sniffing disabled (nosniff).")
	} else {
		rep.add("X-Content-Type-Options", false, xcto, GradeMissing, "A05 Security Misconfiguration",
			"Set to nosniff so browsers don't MIME-sniff responses.")
	}

	// Referrer-Policy.
	ref := get("Referrer-Policy")
	if ref != "" {
		rep.add("Referrer-Policy", true, ref, GradeOK, "A01 Broken Access Control",
			"Referrer-Policy set — limits what referrer data leaks to other sites.")
	} else {
		rep.add("Referrer-Policy", false, "", GradeMissing, "A01 Broken Access Control",
			"No Referrer-Policy — full URLs may leak to third parties. Use no-referrer or strict-origin-when-cross-origin.")
	}

	// Permissions-Policy.
	pp := get("Permissions-Policy")
	if pp != "" {
		rep.add("Permissions-Policy", true, pp, GradeOK, "A05 Security Misconfiguration",
			"Permissions-Policy set — restricts powerful browser features.")
	} else {
		rep.add("Permissions-Policy", false, "", GradeMissing, "A05 Security Misconfiguration",
			"No Permissions-Policy — camera, geolocation, etc. aren't restricted at the header level.")
	}

	// Information disclosure (headers that should be absent or generic).
	if s := get("Server"); versionLike(s) {
		rep.add("Server", true, s, GradeInfo, "A05 Security Misconfiguration",
			"Server header leaks the software version — makes targeting known CVEs easier. Suppress or generalise it.")
	}
	if xp := get("X-Powered-By"); xp != "" {
		rep.add("X-Powered-By", true, xp, GradeInfo, "A05 Security Misconfiguration",
			"X-Powered-By discloses the stack/version — remove it.")
	}

	rep.score()
	return rep
}

func (r *Report) add(name string, present bool, value, grade, owasp, note string) {
	r.Headers = append(r.Headers, Header{Name: name, Present: present, Value: value, Grade: grade, Note: note, OWASP: owasp})
}

// score computes a 0–100 score and A–F grade from the graded headers.
func (r *Report) score() {
	weights := map[string]int{
		"Strict-Transport-Security": 25,
		"Content-Security-Policy":   25,
		"X-Frame-Options":           15,
		"X-Content-Type-Options":    15,
		"Referrer-Policy":           10,
		"Permissions-Policy":        10,
	}
	s := 0
	for _, hd := range r.Headers {
		switch hd.Grade {
		case GradeOK:
			s += weights[hd.Name]
		case GradeWarn:
			s += weights[hd.Name] / 2
		case GradeInfo:
			s -= 10
		}
	}
	if s < 0 {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	r.Score = s
	switch {
	case s >= 90:
		r.Grade = "A"
	case s >= 75:
		r.Grade = "B"
	case s >= 55:
		r.Grade = "C"
	case s >= 35:
		r.Grade = "D"
	default:
		r.Grade = "F"
	}
}

var maxAgeRe = regexp.MustCompile(`(?i)max-age\s*=\s*(\d+)`)

func maxAge(hsts string) int {
	m := maxAgeRe.FindStringSubmatch(hsts)
	if m == nil {
		return 0
	}
	n := 0
	for _, c := range m[1] {
		n = n*10 + int(c-'0')
		if n > 1<<40 {
			break
		}
	}
	return n
}

func versionLike(s string) bool {
	return strings.ContainsAny(s, "0123456789") && regexp.MustCompile(`\d+\.\d`).MatchString(s)
}
