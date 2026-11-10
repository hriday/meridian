package lookup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const rdapDomainBody = `{
  "ldhName":"cloudflare.com",
  "nameservers":[{"ldhName":"ns3.cloudflare.com"},{"ldhName":"ns4.cloudflare.com"}],
  "secureDNS":{"delegationSigned":true},
  "events":[
    {"eventAction":"registration","eventDate":"2009-02-17T22:07:54Z"},
    {"eventAction":"expiration","eventDate":"2026-02-17T00:00:00Z"}
  ],
  "entities":[{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]
}`

func TestRdapDomainParses(t *testing.T) {
	c := rdapServer(t, rdapDomainBody)
	reg, err := c.rdapDomain(context.Background(), "cloudflare.com")
	require.NoError(t, err)
	require.Equal(t, "Cloudflare, Inc.", reg.Registrar)
	require.Contains(t, reg.Created, "2009-02-17")
	require.Contains(t, reg.Expires, "2026-02-17")
	require.Equal(t, []string{"ns3.cloudflare.com", "ns4.cloudflare.com"}, reg.Nameservers)
	require.Equal(t, "signed", reg.DNSSEC)
}

// A fuller RDAP body carrying the Whois-grade extras: status codes, a
// last-changed event, and a registrar with a nested abuse entity.
const rdapDomainFullBody = `{
  "ldhName":"cloudflare.com",
  "status":["clientTransferProhibited","clientDeleteProhibited"],
  "nameservers":[{"ldhName":"ns3.cloudflare.com"}],
  "secureDNS":{"delegationSigned":true},
  "events":[
    {"eventAction":"registration","eventDate":"2009-02-17T22:07:54Z"},
    {"eventAction":"expiration","eventDate":"2026-02-17T00:00:00Z"},
    {"eventAction":"last changed","eventDate":"2024-05-01T00:00:00Z"}
  ],
  "entities":[{
    "roles":["registrar"],
    "vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]],
    "entities":[{"roles":["abuse"],"vcardArray":["vcard",[["email",{},"text","abuse@registrar.example"]]]}]
  }]
}`

func TestRdapDomainParsesWhoisFields(t *testing.T) {
	c := rdapServer(t, rdapDomainFullBody)
	reg, err := c.rdapDomain(context.Background(), "cloudflare.com")
	require.NoError(t, err)
	require.Equal(t, []string{"clientTransferProhibited", "clientDeleteProhibited"}, reg.Statuses)
	require.Contains(t, reg.Updated, "2024-05-01")
	require.Equal(t, "abuse@registrar.example", reg.AbuseEmail)
	// The original fields still parse.
	require.Equal(t, "Cloudflare, Inc.", reg.Registrar)
	require.Contains(t, reg.Created, "2009-02-17")
	require.Contains(t, reg.Expires, "2026-02-17")
}

func TestRdapDomainMissingWhoisFieldsEmpty(t *testing.T) {
	// The original body has no status, no last-changed, no abuse entity.
	c := rdapServer(t, rdapDomainBody)
	reg, err := c.rdapDomain(context.Background(), "cloudflare.com")
	require.NoError(t, err)
	require.Empty(t, reg.Statuses)
	require.Equal(t, "", reg.Updated)
	require.Equal(t, "", reg.AbuseEmail)
}
