package lookup

import (
	"context"
	"time"

	"github.com/hriday/meridian/internal/asnmap"
	"github.com/hriday/meridian/internal/dnsx"
)

// DNSQuery is the DNS query seam (tests inject a fake); dnsServer is the
// resolver domain lookups use.
var (
	DNSQuery  = dnsx.Query
	dnsServer = "1.1.1.1"
)

var domainRecordTypes = []string{"A", "AAAA", "MX", "NS", "TXT"}

// DomainReport is the "where does this domain resolve / who's it registered to"
// answer: DNS records, the IPs it resolves to and who hosts them, and the RDAP
// registration (empty when the TLD serves no RDAP).
type DomainReport struct {
	Input       string        `json:"input"`
	Rcode       string        `json:"rcode"` // A-record query status (NOERROR / NXDOMAIN / …)
	Records     []dnsx.Record `json:"records"`
	ResolvesTo  []string      `json:"resolvesTo"`
	HostedBy    []string      `json:"hostedBy"`
	Registrar   string        `json:"registrar"`
	Created     string        `json:"created"`
	Updated     string        `json:"updated"`
	Expires     string        `json:"expires"`
	DNSSEC      string        `json:"dnssec"`
	Statuses    []string      `json:"statuses"`
	Nameservers []string      `json:"nameservers"`
	AbuseEmail  string        `json:"abuseEmail"`
	DNSSrc      Source        `json:"dnsSrc"`
	RegistrySrc Source        `json:"registrySrc"`
}

// LookupDomain resolves a domain's records (via the DNS engine), the networks
// hosting its addresses (via Team Cymru), and its RDAP registration.
func (c *Client) LookupDomain(ctx context.Context, name string) (DomainReport, error) {
	// Bound the whole lookup so a slow/large domain can't stack many record +
	// hosting queries without end.
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	r := DomainReport{Input: name}

	// DNS records.
	r.DNSSrc = Source{Label: "DNS · " + dnsServer, FetchedAt: now()}
	var firstErr error
	for _, rtype := range domainRecordTypes {
		res, err := DNSQuery(ctx, name, rtype, dnsServer)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if rtype == "A" {
			r.Rcode = res.Rcode // the authoritative "does it resolve" status
		}
		r.Records = append(r.Records, res.Answers...)
		if rtype == "A" || rtype == "AAAA" {
			for _, a := range res.Answers {
				r.ResolvesTo = append(r.ResolvesTo, a.Value)
			}
		}
	}
	if len(r.Records) == 0 && firstErr != nil {
		r.DNSSrc.Err = firstErr.Error()
	}

	// Who hosts the resolved addresses (Team Cymru) — each unique IP queried once.
	seenIP := map[string]bool{}
	seenOwner := map[string]bool{}
	for _, ip := range r.ResolvesTo {
		if ip == "" || seenIP[ip] {
			continue
		}
		seenIP[ip] = true
		info, err := asnmap.Lookup(ctx, asnmap.QueryFunc(DNSQuery), ip)
		if err != nil || info.Owner == "" || seenOwner[info.Owner] {
			continue
		}
		seenOwner[info.Owner] = true
		r.HostedBy = append(r.HostedBy, info.Owner)
	}

	// Registration (RDAP domain) — not served by every TLD.
	r.RegistrySrc = Source{Label: "RDAP", FetchedAt: now()}
	if reg, err := c.rdapDomain(ctx, name); err != nil {
		r.RegistrySrc.Err = err.Error()
	} else {
		r.Registrar, r.Created, r.Expires = reg.Registrar, reg.Created, reg.Expires
		r.Updated, r.DNSSEC, r.Nameservers = reg.Updated, reg.DNSSEC, reg.Nameservers
		r.Statuses, r.AbuseEmail = reg.Statuses, reg.AbuseEmail
	}
	return r, nil
}
