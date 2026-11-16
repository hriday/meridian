package openresolver

import (
	"context"
	"errors"
	"net"
	"testing"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func msgWith(rcode int, ra bool, answer bool) *mdns.Msg {
	m := new(mdns.Msg)
	m.SetQuestion("example.com.", mdns.TypeA)
	m.Response = true
	m.Rcode = rcode
	m.RecursionAvailable = ra
	if answer {
		m.Answer = []mdns.RR{&mdns.A{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 300}, A: net.ParseIP("93.184.216.34")}}
	}
	return m
}

func TestClassify(t *testing.T) {
	open := classify("10.0.0.1", msgWith(mdns.RcodeSuccess, true, true))
	require.True(t, open.Open)
	require.True(t, open.RecursionAvailable)
	require.True(t, open.Answered)
	require.Contains(t, open.Summary, "OPEN")

	refused := classify("10.0.0.2", msgWith(mdns.RcodeRefused, false, false))
	require.False(t, refused.Open)
	require.Contains(t, refused.Summary, "did not recurse")

	// Recursion advertised but the name wasn't resolved — not open.
	noAns := classify("10.0.0.3", msgWith(mdns.RcodeSuccess, true, false))
	require.False(t, noAns.Open)

	// Answered but no RA bit (e.g. an authoritative server) — not open.
	auth := classify("10.0.0.4", msgWith(mdns.RcodeSuccess, false, true))
	require.False(t, auth.Open)

	require.False(t, classify("x", nil).Open)
}

func TestTestInjectsQuery(t *testing.T) {
	defer func(q func(context.Context, string, string, string) (*mdns.Msg, error)) { queryFn = q }(queryFn)

	queryFn = func(_ context.Context, _, _, _ string) (*mdns.Msg, error) {
		return msgWith(mdns.RcodeSuccess, true, true), nil
	}
	rep, err := Test(context.Background(), "10.0.0.1")
	require.NoError(t, err)
	require.True(t, rep.Open)

	queryFn = func(_ context.Context, _, _, _ string) (*mdns.Msg, error) {
		return nil, errors.New("i/o timeout")
	}
	_, err = Test(context.Background(), "10.0.0.9")
	require.Error(t, err, "a transport failure is an error, not a 'closed' verdict")
}
