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
