// Package macvendor finds the vendor of a MAC address from its OUI (the
// IEEE-assigned first three bytes) via a keyless public MAC-vendor API. The
// query carries only the OUI and a generic User-Agent — never who asked.
// normalize is pure; Lookup does the network exchange over an injected Doer so
// tests stay offline. The CLI and Wails controller are thin adapters over Lookup.
package macvendor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// userAgent identifies Meridian generically, with no user identity.
const userAgent = "meridian/0.1 (+https://github.com/hriday/meridian)"

// baseURL is the keyless MAC-vendor API root; a package var so tests can point
// it at an httptest server.
var baseURL = "https://api.macvendors.com/"

// Doer performs an HTTP request (satisfied by *http.Client).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Result is the outcome of a vendor lookup.
type Result struct {
	Input               string `json:"input"`
	OUI                 string `json:"oui"`
	Vendor              string `json:"vendor"`
	Found               bool   `json:"found"`
	LocallyAdministered bool   `json:"locallyAdministered"`
}

// normalize validates a MAC/OUI (any common spelling), returning the OUI as
// "AA:BB:CC", the 6-hex query form, and whether the address is
// locally-administered (bit 0x02 of the first byte). It requires at least three
// bytes of hex.
func normalize(input string) (oui, query string, localAdmin bool, err error) {
	s := strings.Map(func(r rune) rune {
		switch r {
		case ':', '-', '.', ' ':
			return -1
		}
		return r
	}, input)
	if len(s) < 6 {
		return "", "", false, fmt.Errorf("need at least a 3-byte OUI (6 hex digits), got %q", input)
	}
	s = s[:6]
	if !isHex(s) {
		return "", "", false, fmt.Errorf("not a hex MAC/OUI: %q", input)
	}
	up := strings.ToUpper(s)
	first, err := hexByte(up[:2])
	if err != nil {
		return "", "", false, err
	}
	return up[0:2] + ":" + up[2:4] + ":" + up[4:6], up, first&0x02 != 0, nil
}

// Lookup normalizes input and fetches its vendor. A bad input errors before any
// request. 404 means the OUI isn't registered (Found=false, nil error); 429 is a
// rate-limit error; other non-2xx and transport failures are errors.
func Lookup(ctx context.Context, d Doer, input string) (Result, error) {
	oui, query, local, err := normalize(input)
	if err != nil {
		return Result{}, err
	}
	res := Result{Input: input, OUI: oui, LocallyAdministered: local}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+query, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))

	switch {
	case resp.StatusCode == http.StatusNotFound:
		res.Found = false
		return res, nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return Result{}, errors.New("the free MAC-vendor API allows about one lookup a second — wait a moment and try again")
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		res.Vendor = strings.TrimSpace(string(body))
		res.Found = res.Vendor != ""
		return res, nil
	default:
		return Result{}, fmt.Errorf("MAC-vendor lookup failed: status %d", resp.StatusCode)
	}
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func hexByte(s string) (byte, error) {
	var v byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d byte
		switch {
		case c >= '0' && c <= '9':
			d = c - '0'
		case c >= 'A' && c <= 'F':
			d = c - 'A' + 10
		case c >= 'a' && c <= 'f':
			d = c - 'a' + 10
		default:
			return 0, fmt.Errorf("bad hex %q", s)
		}
		v = v<<4 | d
	}
	return v, nil
}
