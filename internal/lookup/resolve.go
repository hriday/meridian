package lookup

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"
)

// IPReport is the "who owns this" answer for an IP address. Each data section
// carries its own Source; a section whose Source.Err is non-empty failed while
// the rest of the report still stands.
type IPReport struct {
	Input          string `json:"input"`
	Version        string `json:"version"` // "IPv4" or "IPv6"
	CoveringPrefix string `json:"coveringPrefix"`
	OriginASN      string `json:"originAsn"`  // e.g. "AS13335", "" if unannounced
	OriginName     string `json:"originName"` // holder of the origin ASN
	RegisteredOrg  string `json:"registeredOrg"`
	AbuseEmail     string `json:"abuseEmail"`
	Registry       string `json:"registry"`    // RIR name (ARIN, RIPE NCC, …)
	RPKIState      string `json:"rpkiState"`   // valid/invalid/unknown ("" if not checked)
	RoutingNote    string `json:"routingNote"` // set for non-global addresses
	RoutingSrc     Source `json:"routingSrc"`
	RegistrySrc    Source `json:"registrySrc"`
}

// ASNReport is the report for an autonomous system.
type ASNReport struct {
	ASN         string `json:"asn"`
	Holder      string `json:"holder"`
	Registry    string `json:"registry"`
	V4Prefixes  int    `json:"v4Prefixes"`
	V6Prefixes  int    `json:"v6Prefixes"`
	Neighbours  int    `json:"neighbours"`
	RoutingSrc  Source   `json:"routingSrc"`
	RegistrySrc Source   `json:"registrySrc"`
	Peering     *Peering `json:"peering,omitempty"`
	PeeringSrc  Source   `json:"peeringSrc"`
}

func now() time.Time { return time.Now().UTC() }

// LookupIP resolves an IP to its owner. Non-global addresses (private, loopback,
// link-local, multicast) are answered locally without any network call.
func (c *Client) LookupIP(ctx context.Context, ip string) (IPReport, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return IPReport{}, fmt.Errorf("not a valid IP address: %w", err)
	}
	r := IPReport{Input: addr.String(), Version: "IPv4"}
	if addr.Is6() {
		r.Version = "IPv6"
	}

	if note := nonGlobalNote(addr); note != "" {
		r.RoutingNote = note
		r.RoutingSrc = Source{Label: "local", FetchedAt: now()}
		r.RegistrySrc = Source{Label: "—", FetchedAt: now()}
		return r, nil
	}

	// Routing (RIPEstat).
	r.RoutingSrc = Source{Label: "RIPEstat", FetchedAt: now()}
	if prefix, asns, err := c.ripeNetworkInfo(ctx, addr.String()); err != nil {
		r.RoutingSrc.Err = err.Error()
	} else {
		r.CoveringPrefix = prefix
		if len(asns) > 0 {
			r.OriginASN = "AS" + asns[0]
			if holder, err := c.ripeASOverview(ctx, r.OriginASN); err == nil {
				r.OriginName = holder
			}
			if prefix != "" {
				if state, _, err := c.ripeRPKI(ctx, prefix, r.OriginASN); err == nil {
					r.RPKIState = state
				}
			}
		}
	}

	// Registry / ownership (RDAP).
	r.RegistrySrc = Source{Label: "RDAP", FetchedAt: now()}
	if e, err := c.rdapIP(ctx, addr.String()); err != nil {
		r.RegistrySrc.Err = err.Error()
	} else {
		r.RegisteredOrg = e.Org
		r.AbuseEmail = e.AbuseEmail
		r.Registry = e.Registry
		if e.Registry != "" {
			r.RegistrySrc.Label = "RDAP · " + e.Registry
		}
	}
	return r, nil
}

// LookupASN resolves an ASN (accepts "13335" or "AS13335").
func (c *Client) LookupASN(ctx context.Context, asn string) (ASNReport, error) {
	num := strings.TrimLeft(strings.ToUpper(strings.TrimSpace(asn)), "AS")
	if num == "" || strings.IndexFunc(num, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ASNReport{}, fmt.Errorf("not a valid ASN: %q", asn)
	}
	r := ASNReport{ASN: "AS" + num}

	r.RoutingSrc = Source{Label: "RIPEstat", FetchedAt: now()}
	if v4, v6, nb, err := c.ripeRoutingStatus(ctx, r.ASN); err != nil {
		r.RoutingSrc.Err = err.Error()
	} else {
		r.V4Prefixes, r.V6Prefixes, r.Neighbours = v4, v6, nb
	}
	if holder, err := c.ripeASOverview(ctx, r.ASN); err == nil {
		r.Holder = holder
	}

	r.RegistrySrc = Source{Label: "RDAP", FetchedAt: now()}
	if e, err := c.rdapASN(ctx, r.ASN); err != nil {
		r.RegistrySrc.Err = err.Error()
	} else {
		r.Registry = e.Registry
		if r.Holder == "" {
			r.Holder = e.Org
		}
		if e.Registry != "" {
			r.RegistrySrc.Label = "RDAP · " + e.Registry
		}
	}

	// Peering (PeeringDB) — best-effort enrichment; absent when the ASN isn't
	// in PeeringDB, which is not an error.
	r.PeeringSrc = Source{Label: "PeeringDB", FetchedAt: now()}
	if p, err := c.peeringDB(ctx, r.ASN); err != nil {
		r.PeeringSrc.Err = err.Error()
	} else {
		r.Peering = p
	}
	return r, nil
}

// reservedPrefixes are special-use ranges that netip's predicates don't cover
// but that are never globally routed, so a lookup should answer locally.
var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),      // "this network" (RFC 1122)
	netip.MustParsePrefix("100.64.0.0/10"),  // CGNAT (RFC 6598)
	netip.MustParsePrefix("192.0.0.0/24"),   // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),   // TEST-NET-1 (documentation)
	netip.MustParsePrefix("198.18.0.0/15"),  // benchmarking (RFC 2544)
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2 (documentation)
	netip.MustParsePrefix("203.0.113.0/24"), // TEST-NET-3 (documentation)
	netip.MustParsePrefix("240.0.0.0/4"),    // reserved (RFC 1112)
	netip.MustParsePrefix("2001:db8::/32"),  // IPv6 documentation (RFC 3849)
}

// nonGlobalNote returns a human note if addr is not a globally routable unicast
// address (so routing/registry lookups would be meaningless), else "".
func nonGlobalNote(a netip.Addr) string {
	switch {
	case a.IsLoopback():
		return "loopback address — not globally routed"
	case a.IsPrivate():
		return "private address (RFC 1918 / ULA) — not globally routed"
	case a.IsLinkLocalUnicast(), a.IsLinkLocalMulticast():
		return "link-local address — not globally routed"
	case a.IsMulticast():
		return "multicast address — not globally routed"
	case a.IsUnspecified():
		return "unspecified address — not globally routed"
	}
	for _, p := range reservedPrefixes {
		if p.Contains(a) {
			return "special-use / reserved address — not globally routed"
		}
	}
	return ""
}
