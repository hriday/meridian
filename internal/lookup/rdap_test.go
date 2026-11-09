package lookup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func rdapServer(t *testing.T, body string) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	old := rdapBase
	rdapBase = srv.URL + "/"
	t.Cleanup(func() { rdapBase = old })
	return NewClient(http.DefaultClient, time.Minute)
}

const arinIPBody = `{
  "handle":"NET-104-16-0-0-1","name":"CLOUDFLARENET","port43":"whois.arin.net",
  "entities":[
    {"handle":"CLOUDFLARENET","roles":["registrant"],
     "vcardArray":["vcard",[["version",{},"text","4.0"],["fn",{},"text","Cloudflare, Inc."]]],
     "entities":[
       {"handle":"ABUSE","roles":["abuse"],
        "vcardArray":["vcard",[["fn",{},"text","Abuse"],["email",{},"text","abuse@cloudflare.com"]]]}
     ]}
  ]}`

func TestRdapIPExtractsOrgAndAbuse(t *testing.T) {
	c := rdapServer(t, arinIPBody)
	e, err := c.rdapIP(context.Background(), "104.16.123.45")
	require.NoError(t, err)
	require.Equal(t, "Cloudflare, Inc.", e.Org)
	require.Equal(t, "abuse@cloudflare.com", e.AbuseEmail)
	require.Equal(t, "ARIN", e.Registry)
	require.Equal(t, "NET-104-16-0-0-1", e.Handle)
}

// A RIPE-style body whose registrant has no vCard and that carries no abuse
// entity must still parse: handle + a best-effort org, empty abuse, no error.
const ripeIPBody = `{
  "handle":"185.0.0.0 - 185.255.255.255","name":"EU-ZONE","port43":"whois.ripe.net",
  "entities":[{"handle":"ORG-XY","roles":["registrant"]}]}`

func TestRdapToleratesMissingVCard(t *testing.T) {
	c := rdapServer(t, ripeIPBody)
	e, err := c.rdapIP(context.Background(), "185.1.2.3")
	require.NoError(t, err)
	require.Equal(t, "185.0.0.0 - 185.255.255.255", e.Handle)
	require.Equal(t, "", e.AbuseEmail)
	require.Equal(t, "EU-ZONE", e.Org) // fell back to top-level name
	require.Contains(t, e.Registry, "RIPE")
}

const arinASNBody = `{
  "handle":"AS13335","name":"CLOUDFLARENET","port43":"whois.arin.net",
  "entities":[{"roles":["registrant"],
    "vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]}`

func TestRdapASN(t *testing.T) {
	c := rdapServer(t, arinASNBody)
	e, err := c.rdapASN(context.Background(), "AS13335")
	require.NoError(t, err)
	require.Equal(t, "AS13335", e.Handle)
	require.Equal(t, "Cloudflare, Inc.", e.Org)
	require.True(t, strings.Contains(e.Registry, "ARIN"))
}
