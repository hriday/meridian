package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/lookup"
	"github.com/stretchr/testify/require"
)

const rdapDomainBody = `{"ldhName":"cloudflare.com","nameservers":[{"ldhName":"ns3.cloudflare.com"}],"secureDNS":{"delegationSigned":true},"events":[{"eventAction":"registration","eventDate":"2009-02-17T00:00:00Z"}],"entities":[{"roles":["registrar"],"vcardArray":["vcard",[["fn",{},"text","Cloudflare, Inc."]]]}]}`

type domainDoer struct{}

func (domainDoer) Do(r *http.Request) (*http.Response, error) {
	body := `{}`
	if strings.Contains(r.URL.Path, "/domain/") {
		body = rdapDomainBody
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
}

func TestAppLookupDomain(t *testing.T) {
	oldQ := lookup.DNSQuery
	t.Cleanup(func() { lookup.DNSQuery = oldQ })
	lookup.DNSQuery = func(_ context.Context, name, rtype, _ string) (dnsx.Result, error) {
		if strings.Contains(name, "cymru") {
			return dnsx.Result{Answers: []dnsx.Record{{Type: "TXT", Value: `"13335 | 1.0.0.0/8 | US | arin | x | CLOUDFLARENET, US"`}}}, nil
		}
		if rtype == "A" {
			return dnsx.Result{Rcode: "NOERROR", Answers: []dnsx.Record{{Name: name, Type: "A", Value: "104.16.132.229"}}}, nil
		}
		return dnsx.Result{Rcode: "NOERROR"}, nil
	}

	a := New()
	a.lookup = lookup.NewClient(domainDoer{}, time.Minute)
	r, err := a.LookupDomain("cloudflare.com")
	require.NoError(t, err)
	require.Equal(t, "Cloudflare, Inc.", r.Registrar)
	require.NotEmpty(t, r.Records)
	require.Contains(t, r.ResolvesTo, "104.16.132.229")
}
