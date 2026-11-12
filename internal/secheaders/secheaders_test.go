package secheaders

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func byName(r Report) map[string]Header {
	m := map[string]Header{}
	for _, h := range r.Headers {
		m[h.Name] = h
	}
	return m
}

func TestClassifyAllGood(t *testing.T) {
	h := http.Header{}
	h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	h.Set("Content-Security-Policy", "default-src 'self'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Permissions-Policy", "geolocation=()")

	r := classify(h, "https://example.com/", 200)
	m := byName(r)
	require.Equal(t, GradeOK, m["Strict-Transport-Security"].Grade)
	require.Equal(t, GradeOK, m["Content-Security-Policy"].Grade)
	require.Equal(t, "A", r.Grade)
	require.GreaterOrEqual(t, r.Score, 90)
	// OWASP tags are present.
	require.Contains(t, m["Content-Security-Policy"].OWASP, "A03")
}

func TestClassifyAllMissing(t *testing.T) {
	r := classify(http.Header{}, "https://example.com/", 200)
	m := byName(r)
	require.Equal(t, GradeMissing, m["Strict-Transport-Security"].Grade)
	require.Equal(t, GradeMissing, m["Content-Security-Policy"].Grade)
	require.Equal(t, "F", r.Grade)
	require.Equal(t, 0, r.Score)
}

func TestClassifyWeakVariants(t *testing.T) {
	h := http.Header{}
	h.Set("Strict-Transport-Security", "max-age=3600") // short
	h.Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline'")
	r := classify(h, "https://example.com/", 200)
	m := byName(r)
	require.Equal(t, GradeWarn, m["Strict-Transport-Security"].Grade)
	require.Equal(t, GradeWarn, m["Content-Security-Policy"].Grade)
}

func TestClassifyFrameAncestorsCountsForClickjacking(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Security-Policy", "frame-ancestors 'none'")
	r := classify(h, "https://example.com/", 200)
	require.Equal(t, GradeOK, byName(r)["X-Frame-Options"].Grade)
}

func TestClassifyNonHTTPSHSTSWarn(t *testing.T) {
	r := classify(http.Header{}, "http://example.com/", 200)
	require.Equal(t, GradeWarn, byName(r)["Strict-Transport-Security"].Grade)
	require.False(t, r.HTTPS)
}

func TestClassifyServerVersionDisclosure(t *testing.T) {
	h := http.Header{}
	h.Set("Server", "Apache/2.4.49 (Unix)")
	h.Set("X-Powered-By", "PHP/7.4.3")
	r := classify(h, "https://example.com/", 200)
	m := byName(r)
	require.Equal(t, GradeInfo, m["Server"].Grade)
	require.Equal(t, GradeInfo, m["X-Powered-By"].Grade)

	// A generic Server header (no version) isn't flagged.
	h2 := http.Header{}
	h2.Set("Server", "cloudflare")
	require.NotContains(t, byName(classify(h2, "https://example.com/", 200)), "Server")
}

func TestMaxAge(t *testing.T) {
	require.Equal(t, 31536000, maxAge("max-age=31536000; includeSubDomains"))
	require.Equal(t, 0, maxAge("includeSubDomains"))
}
