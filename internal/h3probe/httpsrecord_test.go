package h3probe

import (
	"context"
	"errors"
	"net"
	"testing"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestParseSVCB(t *testing.T) {
	params := []mdns.SVCBKeyValue{
		&mdns.SVCBAlpn{Alpn: []string{"h3", "h2"}},
		&mdns.SVCBPort{Port: 8443},
		&mdns.SVCBIPv4Hint{Hint: []net.IP{net.ParseIP("1.2.3.4")}},
		&mdns.SVCBIPv6Hint{Hint: []net.IP{net.ParseIP("2606:4700::1")}},
		&mdns.SVCBECHConfig{ECH: []byte{0xAB, 0xCD}},
	}
	r := parseSVCB(1, "svc.example.com.", params)
	require.True(t, r.Found)
	require.True(t, r.AdvertisesH3, "ALPN contains h3")
	require.Equal(t, []string{"h3", "h2"}, r.ALPN)
	require.Equal(t, 8443, r.Port)
	require.Equal(t, "svc.example.com", r.Target, "trailing dot trimmed")
	require.Contains(t, r.IPv4Hint, "1.2.3.4")
	require.Contains(t, r.IPv6Hint, "2606:4700::1")
	require.True(t, r.ECH, "an ECH config is present")
}

func TestParseSVCBNoH3NoECH(t *testing.T) {
	r := parseSVCB(1, ".", []mdns.SVCBKeyValue{&mdns.SVCBAlpn{Alpn: []string{"h2"}}})
	require.True(t, r.Found)
	require.False(t, r.AdvertisesH3)
	require.False(t, r.ECH)
}

func TestLookupHTTPS(t *testing.T) {
	defer func(q func(context.Context, string, string, string) (*mdns.Msg, error)) { httpsQueryFn = q }(httpsQueryFn)

	httpsQueryFn = func(_ context.Context, _, _, _ string) (*mdns.Msg, error) {
		m := new(mdns.Msg)
		m.Answer = []mdns.RR{&mdns.HTTPS{SVCB: mdns.SVCB{
			Hdr:      mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeHTTPS},
			Priority: 1, Target: ".",
			Value: []mdns.SVCBKeyValue{&mdns.SVCBAlpn{Alpn: []string{"h3", "h2"}}, &mdns.SVCBECHConfig{ECH: []byte{1}}},
		}}}
		return m, nil
	}
	r := lookupHTTPS(context.Background(), "example.com")
	require.NotNil(t, r)
	require.True(t, r.Found)
	require.True(t, r.AdvertisesH3)
	require.True(t, r.ECH)

	// A query error is nil (not a false "no record").
	httpsQueryFn = func(_ context.Context, _, _, _ string) (*mdns.Msg, error) {
		return nil, errors.New("timeout")
	}
	require.Nil(t, lookupHTTPS(context.Background(), "example.com"))

	// A successful query with no HTTPS record is Found:false.
	httpsQueryFn = func(_ context.Context, _, _, _ string) (*mdns.Msg, error) {
		return new(mdns.Msg), nil
	}
	r = lookupHTTPS(context.Background(), "example.com")
	require.NotNil(t, r)
	require.False(t, r.Found)
}
