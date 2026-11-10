// Package dnsbl checks whether an IP address is listed on public DNS-based
// blocklists (DNSBLs / RBLs). For each blocklist zone it queries the reversed
// IP under that zone and classifies the answer: a 127.0.0.0/8 address means
// listed, NXDOMAIN means clean, and the 127.255.255.0/24 band means the query
// was refused (public/open resolver, a typo, or rate-limiting) — an error, not
// a listing, which callers must never read as "clean" or "listed".
//
// reverseName and classify are pure; Check does the network exchange via the
// dnsx seam. The CLI and Wails controller are thin adapters over Check.
package dnsbl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/dnsx"
	mdns "github.com/miekg/dns"
)

// Status is the per-zone outcome of a blocklist lookup.
type Status string

const (
	StatusListed  Status = "listed"  // the IP is on this blocklist
	StatusClean   Status = "clean"   // the IP is not on this blocklist
	StatusBlocked Status = "blocked" // the query was refused (resolver/volume/typo)
	StatusError   Status = "error"   // the query failed or returned a non-DNSBL answer
)

// Zone is a blocklist to check: its DNS zone, a human label, and an optional
// note (e.g. that the list needs the user's resolver IP registered).
type Zone struct {
	Host string `json:"host"`
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
}

// DefaultZones is a small, well-known starting set of public IP blocklists. The
// engine is list-agnostic: a defunct or registration-gated zone reports an
// error/blocked status honestly rather than a false "clean".
func DefaultZones() []Zone {
	return []Zone{
		{Host: "zen.spamhaus.org", Name: "Spamhaus ZEN"},
		{Host: "bl.spamcop.net", Name: "SpamCop"},
		{Host: "dnsbl.dronebl.org", Name: "DroneBL"},
		{Host: "b.barracudacentral.org", Name: "Barracuda", Note: "requires registering your resolver IP"},
	}
}

// reverseName builds the DNSBL query name for ip under zone: the reversed IPv4
// octets or the 32 reversed IPv6 nibbles, followed by the zone. It errors on a
// non-IP input (this tool is IP-only).
func reverseName(ip, zone string) (string, error) {
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("not an IP address: %q (resolve a hostname first)", ip)
	}
	rev, err := mdns.ReverseAddr(ip)
	if err != nil {
		return "", err
	}
	rev = strings.TrimSuffix(rev, ".in-addr.arpa.")
	rev = strings.TrimSuffix(rev, ".ip6.arpa.")
	return rev + "." + zone, nil
}

var (
	loopback8   = mustCIDR("127.0.0.0/8")
	errorBand24 = mustCIDR("127.255.255.0/24")
)

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

// classify maps the A-record answer IPs of a DNSBL query to a status and a
// plain-language note. The error band (127.255.255.0/24) is checked before the
// listing range so a refused query is never read as a listing.
func classify(codes []string) (Status, string) {
	if len(codes) == 0 {
		return StatusClean, ""
	}
	var sawListed, sawBlocked, sawOdd bool
	for _, c := range codes {
		ip := net.ParseIP(c)
		switch {
		case ip != nil && errorBand24.Contains(ip):
			sawBlocked = true
		case ip != nil && loopback8.Contains(ip):
			sawListed = true
		default:
			sawOdd = true
		}
	}
	switch {
	case sawListed:
		return StatusListed, ""
	case sawBlocked:
		return StatusBlocked, "The blocklist refused the query — use your own resolver rather than a big public one (8.8.8.8 / 1.1.1.1), and keep query volume modest."
	case sawOdd:
		return StatusError, "The blocklist returned an address outside 127.0.0.0/8 — not a valid listing response."
	default:
		return StatusClean, ""
	}
}

// queryFn is the DNS exchange, behind a package var so tests can point it at a
// fake. The default reuses the dnsx dig-like client.
var queryFn = dnsx.Query

// Listing is one blocklist's result for the checked IP.
type Listing struct {
	Zone   string   `json:"zone"`
	Name   string   `json:"name"`
	Status Status   `json:"status"`
	Codes  []string `json:"codes,omitempty"`
	Reason string   `json:"reason,omitempty"`
	Note   string   `json:"note,omitempty"`
}

// Report is the aggregate blocklist result for one IP.
type Report struct {
	IP          string    `json:"ip"`
	Resolver    string    `json:"resolver"`
	Checked     int       `json:"checked"`
	Listed      int       `json:"listed"`
	Unreachable int       `json:"unreachable"`
	Verdict     string    `json:"verdict"`
	Summary     string    `json:"summary"`
	Listings    []Listing `json:"listings"`
}

// Check looks up ip against each zone via resolver and returns an aggregate
// report. It errors if ip is not an IP address or resolver is empty. A per-zone
// network failure is recorded as StatusError rather than failing the whole run,
// so one dead list doesn't hide the others. A listed zone gets a best-effort
// TXT reason lookup.
func Check(ctx context.Context, ip, resolver string, zones []Zone, timeout time.Duration) (Report, error) {
	if net.ParseIP(ip) == nil {
		return Report{}, fmt.Errorf("not an IP address: %q (resolve a hostname first)", ip)
	}
	if strings.TrimSpace(resolver) == "" {
		return Report{}, errors.New("no resolver specified")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rep := Report{IP: ip, Resolver: resolver}
	for _, z := range zones {
		l := Listing{Zone: z.Host, Name: z.Name, Note: z.Note}
		name, err := reverseName(ip, z.Host)
		if err != nil {
			return Report{}, err
		}
		res, err := queryFn(ctx, name, "A", resolver)
		if err != nil {
			l.Status = StatusError
			l.Note = err.Error()
			rep.Listings = append(rep.Listings, l)
			continue
		}
		var codes []string
		for _, a := range res.Answers {
			if a.Type == "A" {
				codes = append(codes, a.Value)
			}
		}
		l.Codes = codes
		st, note := classify(codes)
		l.Status = st
		if note != "" {
			l.Note = note
		}
		if st == StatusListed {
			if txt, terr := queryFn(ctx, name, "TXT", resolver); terr == nil {
				var reasons []string
				for _, a := range txt.Answers {
					if a.Type == "TXT" {
						reasons = append(reasons, strings.Trim(a.Value, `"`))
					}
				}
				l.Reason = strings.Join(reasons, " ")
			}
		}
		rep.Listings = append(rep.Listings, l)
	}

	rep.Checked = len(rep.Listings)
	for _, l := range rep.Listings {
		switch l.Status {
		case StatusListed:
			rep.Listed++
		case StatusBlocked, StatusError:
			rep.Unreachable++
		}
	}
	rep.Verdict, rep.Summary = summarize(rep)
	return rep, nil
}

// summarize builds the plain-language verdict and summary. A partial check
// (some zones unreachable) is marked incomplete so "not listed" on the lists
// that answered is never read as a full all-clear.
func summarize(r Report) (verdict, summary string) {
	var listed []string
	for _, l := range r.Listings {
		if l.Status == StatusListed {
			listed = append(listed, l.Name)
		}
	}
	reachable := r.Checked - r.Unreachable
	if r.Listed > 0 {
		verdict = fmt.Sprintf("Listed on %d blocklist(s)", r.Listed)
		summary = "Listed on " + strings.Join(listed, ", ") + "."
	} else {
		verdict = "Not listed"
		summary = fmt.Sprintf("Not listed on any of the %d list(s) that answered.", reachable)
	}
	if r.Unreachable > 0 {
		summary += fmt.Sprintf(" Incomplete: %d of %d list(s) could not be reached (query refused or timed out), so this is not a full all-clear.", r.Unreachable, r.Checked)
	}
	return verdict, summary
}
