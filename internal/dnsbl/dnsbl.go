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
	"fmt"
	"net"
	"strings"

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
