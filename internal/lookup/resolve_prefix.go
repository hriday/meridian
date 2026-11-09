package lookup

import (
	"context"
	"fmt"
	"net/netip"
)

// PrefixReport is the "who announces this prefix" answer.
type PrefixReport struct {
	Input         string   `json:"input"`
	OriginASN     string   `json:"originAsn"`
	OriginName    string   `json:"originName"`
	Announced     bool     `json:"announced"`
	Registry      string   `json:"registry"`
	RegisteredOrg string   `json:"registeredOrg"`
	AbuseEmail    string   `json:"abuseEmail"`
	RPKIState     string   `json:"rpkiState"`
	RPKIRoa       string   `json:"rpkiRoa"`
	LessSpecific  []string `json:"lessSpecific"`
	MoreSpecific  []string `json:"moreSpecific"`
	RoutingSrc    Source   `json:"routingSrc"`
	RegistrySrc   Source   `json:"registrySrc"`
	RPKISrc       Source   `json:"rpkiSrc"`
}

// LookupPrefix resolves a CIDR block: its origin AS, RPKI validity, registry,
// and related (less/more-specific) prefixes.
func (c *Client) LookupPrefix(ctx context.Context, prefix string) (PrefixReport, error) {
	p, err := netip.ParsePrefix(prefix)
	if err != nil {
		return PrefixReport{}, fmt.Errorf("not a valid CIDR: %w", err)
	}
	p = p.Masked()
	r := PrefixReport{Input: p.String()}

	// Routing (RIPEstat): origin + related prefixes.
	r.RoutingSrc = Source{Label: "RIPEstat", FetchedAt: now()}
	if origins, holder, announced, err := c.ripePrefixOverview(ctx, r.Input); err != nil {
		r.RoutingSrc.Err = err.Error()
	} else {
		r.Announced = announced
		r.OriginName = holder
		if len(origins) > 0 {
			r.OriginASN = "AS" + origins[0]
		}
	}
	if less, more, err := c.ripeRelatedPrefixes(ctx, r.Input); err == nil {
		r.LessSpecific, r.MoreSpecific = less, more
	}

	// RPKI (RIPEstat).
	if r.OriginASN != "" {
		r.RPKISrc = Source{Label: "RIPEstat RPKI", FetchedAt: now()}
		if state, roa, err := c.ripeRPKI(ctx, r.Input, r.OriginASN); err != nil {
			r.RPKISrc.Err = err.Error()
		} else {
			r.RPKIState, r.RPKIRoa = state, roa
		}
	}

	// Registry (RDAP) — the block's covering registration.
	r.RegistrySrc = Source{Label: "RDAP", FetchedAt: now()}
	if e, err := c.rdapIP(ctx, r.Input); err != nil {
		r.RegistrySrc.Err = err.Error()
	} else {
		r.RegisteredOrg, r.AbuseEmail, r.Registry = e.Org, e.AbuseEmail, e.Registry
		if e.Registry != "" {
			r.RegistrySrc.Label = "RDAP · " + e.Registry
		}
	}
	return r, nil
}
