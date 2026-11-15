package lookup

import (
	"context"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/whois"
)

// irrWhois is the IRR whois primitive (a seam so tests inject canned RADB text).
var irrWhois = whois.QueryServer

const radbServer = "whois.radb.net"

// IRRRoute is one route/route6 object found in an IRR (the Internet Routing
// Registry): the prefix it authorizes, the AS allowed to originate it, and which
// registry it came from.
type IRRRoute struct {
	Prefix string `json:"prefix"`
	Origin string `json:"origin"` // "AS13335"
	Source string `json:"source"` // RADB, RIPE, ARIN, …
}

// irrRoutes queries RADB (which mirrors the major IRRs) for the route/route6
// objects registered for prefix.
func (c *Client) irrRoutes(ctx context.Context, prefix string) ([]IRRRoute, error) {
	raw, err := irrWhois(ctx, radbServer, prefix, 8*time.Second)
	if err != nil {
		return nil, err
	}
	return parseIRRRoutes(raw), nil
}

// parseIRRRoutes extracts route/route6 objects from an RPSL whois response. A
// route object is a block whose first key is route or route6; blank lines (or
// the next route key) end a block.
func parseIRRRoutes(raw string) []IRRRoute {
	var out []IRRRoute
	var cur IRRRoute
	flush := func() {
		// Exclude RADB's RPKI-derived pseudo route objects (source "RPKI"): they
		// re-expose ROAs, which the separate RPKI check already reports. IRR here
		// means genuine registry route objects.
		if cur.Prefix != "" && !strings.EqualFold(cur.Source, "RPKI") {
			out = append(out, cur)
		}
		cur = IRRRoute{}
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		key, val, ok := splitRPSL(line)
		if !ok {
			continue
		}
		switch key {
		case "route", "route6":
			flush() // a new object begins
			cur.Prefix = val
		case "origin":
			if cur.Prefix != "" {
				cur.Origin = strings.ToUpper(val)
			}
		case "source":
			if cur.Prefix != "" {
				cur.Source = val
			}
		}
	}
	flush()
	return out
}

func splitRPSL(line string) (key, val string, ok bool) {
	i := strings.IndexByte(line, ':')
	if i < 0 {
		return "", "", false
	}
	key = strings.ToLower(strings.TrimSpace(line[:i]))
	val = strings.TrimSpace(line[i+1:])
	if h := strings.IndexByte(val, '#'); h >= 0 { // drop RPSL trailing comment
		val = strings.TrimSpace(val[:h])
	}
	return key, val, true
}

// irrState classifies the route objects against the prefix's announced origins:
// "valid" if some object authorizes an announced origin, "origin-mismatch" if
// objects exist but none match, "not-found" if the IRR has no route object.
func irrState(routes []IRRRoute, origins []string) string {
	if len(routes) == 0 {
		return "not-found"
	}
	want := map[string]bool{}
	for _, o := range origins {
		want[normAS(o)] = true
	}
	for _, rt := range routes {
		if want[normAS(rt.Origin)] {
			return "valid"
		}
	}
	return "origin-mismatch"
}

func normAS(s string) string {
	return strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "AS")
}
