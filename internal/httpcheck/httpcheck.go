// Package httpcheck fetches a URL and reports the operator's-eye view of what an
// endpoint does: the final status, the redirect chain, key response headers, the
// protocol and TLS version, and per-phase timing (DNS, connect, TLS, TTFB,
// total). It is a one-shot metadata check — it reads the response but does not
// download the body. normalizeURL is pure; Check does the request. The CLI and
// Wails controller are thin adapters over Check.
package httpcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	userAgent    = "meridian/0.1 (+https://github.com/hriday/meridian)"
	maxRedirects = 10
)

// Hop is one step in a redirect chain.
type Hop struct {
	Status int    `json:"status"`
	URL    string `json:"url"`
}

// Timing holds per-phase durations in milliseconds.
type Timing struct {
	DNSMillis     float64 `json:"dnsMillis"`
	ConnectMillis float64 `json:"connectMillis"`
	TLSMillis     float64 `json:"tlsMillis"`
	TTFBMillis    float64 `json:"ttfbMillis"`
	TotalMillis   float64 `json:"totalMillis"`
}

// Result is the outcome of an endpoint check.
type Result struct {
	URL           string      `json:"url"`
	FinalURL      string      `json:"finalUrl"`
	Status        int         `json:"status"`
	StatusText    string      `json:"statusText"`
	Proto         string      `json:"proto"`
	TLSVersion    string      `json:"tlsVersion"`
	Server        string      `json:"server"`
	ContentType   string      `json:"contentType"`
	ContentLength int64       `json:"contentLength"`
	Redirects     []Hop       `json:"redirects"`
	Headers       [][2]string `json:"headers"`
	Timing        Timing      `json:"timing"`
}

// newTransport builds the HTTP transport; a package var so tests can inject one
// that trusts an httptest TLS server.
var newTransport = func() *http.Transport {
	return http.DefaultTransport.(*http.Transport).Clone()
}

// normalizeURL adds an https scheme when none is given and rejects empty input.
func normalizeURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("no URL given")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("not a valid URL: %q", raw)
	}
	return s, nil
}

// Check fetches rawURL, following redirects manually so each hop is captured, and
// reports status, headers, protocol, TLS, and timing. A non-2xx final status is
// a result, not an error; a redirect loop past the cap, or a DNS/connect/TLS/
// timeout failure, is an error.
func Check(ctx context.Context, rawURL string, timeout time.Duration) (Result, error) {
	norm, err := normalizeURL(rawURL)
	if err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{
		Transport:     newTransport(),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	res := Result{URL: norm}
	start := time.Now()
	cur := norm

	for i := 0; ; i++ {
		if i > maxRedirects {
			return Result{}, fmt.Errorf("too many redirects (>%d)", maxRedirects)
		}
		resp, timing, err := do(ctx, client, cur)
		if err != nil {
			return Result{}, err
		}
		loc := resp.Header.Get("Location")
		if isRedirect(resp.StatusCode) && loc != "" {
			res.Redirects = append(res.Redirects, Hop{Status: resp.StatusCode, URL: cur})
			next, rerr := resolve(cur, loc)
			resp.Body.Close()
			if rerr != nil {
				return Result{}, rerr
			}
			cur = next
			continue
		}
		// Final response.
		io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		res.FinalURL = cur
		res.Status = resp.StatusCode
		res.StatusText = strings.TrimSpace(strings.TrimPrefix(resp.Status, fmt.Sprintf("%d", resp.StatusCode)))
		res.Proto = resp.Proto
		res.Server = resp.Header.Get("Server")
		res.ContentType = resp.Header.Get("Content-Type")
		res.ContentLength = resp.ContentLength
		res.Headers = headerPairs(resp.Header)
		if resp.TLS != nil {
			res.TLSVersion = tlsName(resp.TLS.Version)
		}
		timing.TotalMillis = ms(time.Since(start))
		res.Timing = timing
		return res, nil
	}
}

// do issues one request with an httptrace capturing phase timings.
func do(ctx context.Context, client *http.Client, u string) (*http.Response, Timing, error) {
	var t Timing
	var dnsStart, connStart, tlsStart, reqStart time.Time
	reqStart = time.Now()
	trace := &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(httptrace.DNSDoneInfo) { t.DNSMillis = ms(time.Since(dnsStart)) },
		ConnectStart:         func(string, string) { connStart = time.Now() },
		ConnectDone:          func(string, string, error) { t.ConnectMillis = ms(time.Since(connStart)) },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { t.TLSMillis = ms(time.Since(tlsStart)) },
		GotFirstResponseByte: func() { t.TTFBMillis = ms(time.Since(reqStart)) },
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, u, nil)
	if err != nil {
		return nil, t, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, t, err
	}
	return resp, t, nil
}

func isRedirect(code int) bool { return code >= 300 && code < 400 }

func resolve(base, loc string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	l, err := url.Parse(loc)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(l).String(), nil
}

func headerPairs(h http.Header) [][2]string {
	out := make([][2]string, 0, len(h))
	for k, vs := range h {
		out = append(out, [2]string{k, strings.Join(vs, ", ")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] }) // stable display order
	return out
}

func tlsName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
