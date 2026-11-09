package lookup

import (
	"context"
	"strings"
)

// DomainReg is the registration slice of a domain's RDAP record.
type DomainReg struct {
	Registrar   string   `json:"registrar"`
	Created     string   `json:"created"`
	Expires     string   `json:"expires"`
	DNSSEC      string   `json:"dnssec"` // "signed" / "unsigned"
	Nameservers []string `json:"nameservers"`
}

// rdapDomain fetches and parses a domain's RDAP record (registrar, dates,
// nameservers, DNSSEC). Tolerant of missing fields; RDAP is not served for
// every TLD, in which case the caller surfaces "no RDAP data".
func (c *Client) rdapDomain(ctx context.Context, name string) (DomainReg, error) {
	var raw struct {
		Nameservers []struct {
			LdhName string `json:"ldhName"`
		} `json:"nameservers"`
		SecureDNS struct {
			DelegationSigned bool `json:"delegationSigned"`
		} `json:"secureDNS"`
		Events []struct {
			Action string `json:"eventAction"`
			Date   string `json:"eventDate"`
		} `json:"events"`
		Entities []rdapEntityRaw `json:"entities"`
	}
	if err := c.GetJSON(ctx, rdapBase+"domain/"+name, &raw); err != nil {
		return DomainReg{}, err
	}

	reg := DomainReg{Registrar: findRole(raw.Entities, "registrar", "fn")}
	for _, e := range raw.Events {
		switch e.Action {
		case "registration":
			reg.Created = e.Date
		case "expiration":
			reg.Expires = e.Date
		}
	}
	for _, ns := range raw.Nameservers {
		if ns.LdhName != "" {
			reg.Nameservers = append(reg.Nameservers, strings.ToLower(ns.LdhName))
		}
	}
	reg.DNSSEC = "unsigned"
	if raw.SecureDNS.DelegationSigned {
		reg.DNSSEC = "signed"
	}
	return reg, nil
}
