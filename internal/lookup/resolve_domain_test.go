package lookup

import (
	"context"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/stretchr/testify/require"
)

func fakeDNS(t *testing.T, aRecord bool) {
	old := DNSQuery
	t.Cleanup(func() { DNSQuery = old })
	DNSQuery = func(_ context.Context, name, rtype, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "cymru") {
			return dnsx.Result{Answers: []dnsx.Record{{Type: "TXT", Value: `"13335 | 104.16.0.0/13 | US | arin | x | CLOUDFLARENET, US"`}}}, nil
		}
		if rtype == "A" && aRecord {
			return dnsx.Result{Rcode: "NOERROR", Answers: []dnsx.Record{{Name: name, Type: "A", TTL: 300, Value: "104.16.132.229"}}}, nil
		}
		return dnsx.Result{Rcode: "NOERROR"}, nil
	}
}

func TestLookupDomainResolvesAndRegisters(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{"/domain/": {body: rdapDomainBody}})
	fakeDNS(t, true)
	r, err := c.LookupDomain(context.Background(), "cloudflare.com")
	require.NoError(t, err)
	require.Equal(t, "Cloudflare, Inc.", r.Registrar)
	require.Contains(t, r.ResolvesTo, "104.16.132.229")
	require.NotEmpty(t, r.HostedBy)
	require.NotEmpty(t, r.Records)
}

func TestLookupDomainNXDOMAIN(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{"/domain/": {code: 404, body: "no rdap"}})
	old := DNSQuery
	t.Cleanup(func() { DNSQuery = old })
	DNSQuery = func(_ context.Context, name, _, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "cymru") {
			return dnsx.Result{}, nil
		}
		return dnsx.Result{Rcode: "NXDOMAIN"}, nil // exists nowhere
	}
	r, err := c.LookupDomain(context.Background(), "nope.example")
	require.NoError(t, err)
	require.Empty(t, r.Records)
	require.Equal(t, "NXDOMAIN", r.Rcode)
	require.Empty(t, r.DNSSrc.Err) // not a DNS failure, a clean "doesn't resolve"
}

func TestLookupDomainDedupsHosting(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{"/domain/": {code: 404, body: ""}})
	old := DNSQuery
	t.Cleanup(func() { DNSQuery = old })
	cymru := 0
	DNSQuery = func(_ context.Context, name, rtype, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "cymru") {
			cymru++
			return dnsx.Result{Answers: []dnsx.Record{{Type: "TXT", Value: `"13335 | 203.0.113.0/24 | US | arin | x | NET, US"`}}}, nil
		}
		if rtype == "A" || rtype == "AAAA" { // A and AAAA both return the SAME address
			return dnsx.Result{Rcode: "NOERROR", Answers: []dnsx.Record{{Name: name, Type: rtype, Value: "203.0.113.9"}}}, nil
		}
		return dnsx.Result{}, nil
	}
	r, err := c.LookupDomain(context.Background(), "x.example")
	require.NoError(t, err)
	require.Len(t, r.HostedBy, 1)
	require.Equal(t, 2, cymru) // one unique IP -> one asnmap.Lookup (origin + AS-name = 2 queries)
}

func TestLookupDomainNoRDAP(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{"/domain/": {code: 404, body: "no rdap"}})
	fakeDNS(t, true)
	r, err := c.LookupDomain(context.Background(), "example.xyz")
	require.NoError(t, err)          // DNS still worked
	require.Empty(t, r.Registrar)    // RDAP unavailable
	require.NotEmpty(t, r.RegistrySrc.Err)
	require.NotEmpty(t, r.Records)
}
