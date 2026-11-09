package lookup

import (
	"context"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/stretchr/testify/require"
)

func fakeDNS(t *testing.T, aRecord bool) {
	old := dnsQueryFn
	t.Cleanup(func() { dnsQueryFn = old })
	dnsQueryFn = func(_ context.Context, name, rtype, _ string) (dnsx.Result, error) {
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

func TestLookupDomainNoRDAP(t *testing.T) {
	c := lookupServer(t, map[string]srvResp{"/domain/": {code: 404, body: "no rdap"}})
	fakeDNS(t, true)
	r, err := c.LookupDomain(context.Background(), "example.xyz")
	require.NoError(t, err)          // DNS still worked
	require.Empty(t, r.Registrar)    // RDAP unavailable
	require.NotEmpty(t, r.RegistrySrc.Err)
	require.NotEmpty(t, r.Records)
}
