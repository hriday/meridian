package app

import (
	"context"
	"testing"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/ntp"
	"github.com/stretchr/testify/require"
)

func TestAppDNSQuery(t *testing.T) {
	old := dnsQueryFn
	t.Cleanup(func() { dnsQueryFn = old })
	dnsQueryFn = func(_ context.Context, name, rtype, server string) (dnsx.Result, error) {
		return dnsx.Result{Rcode: "NOERROR", Answers: []dnsx.Record{{Name: name, Type: rtype, TTL: 300, Value: "1.2.3.4"}}}, nil
	}
	a := New()
	r, err := a.DNSQuery("example.com", "A", "8.8.8.8")
	require.NoError(t, err)
	require.Equal(t, "NOERROR", r.Rcode)
	require.Equal(t, "1.2.3.4", r.Answers[0].Value)
}

func TestAppNTPQuery(t *testing.T) {
	old := ntpQueryFn
	t.Cleanup(func() { ntpQueryFn = old })
	ntpQueryFn = func(_ context.Context, server string) (ntp.Reply, error) {
		return ntp.Reply{Server: server, Stratum: 1, RefID: "GPS"}, nil
	}
	a := New()
	r, err := a.NTPQuery("time.apple.com")
	require.NoError(t, err)
	require.Equal(t, 1, r.Stratum)
}
