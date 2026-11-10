package whois

import (
	"bufio"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// whoisServer is a canned responder: it reads one query line and writes resp.
// It records the last query it received.
type whoisServer struct {
	resp     string
	gotQuery string
}

// fakeNet wires dialWhois to in-memory listeners keyed by host (the "host" part
// of host:43), each backed by a whoisServer.
func fakeNet(t *testing.T, servers map[string]*whoisServer) {
	t.Helper()
	lns := map[string]net.Listener{}
	for host, s := range servers {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		lns[host] = ln
		srv := s
		go func() {
			for {
				c, err := ln.Accept()
				if err != nil {
					return
				}
				line, _ := bufio.NewReader(c).ReadString('\n')
				srv.gotQuery = line
				_, _ = c.Write([]byte(srv.resp))
				_ = c.Close()
			}
		}()
		t.Cleanup(func() { _ = ln.Close() })
	}
	old := dialWhois
	dialWhois = func(ctx context.Context, addr string) (net.Conn, error) {
		host, _, _ := net.SplitHostPort(addr)
		ln, ok := lns[host]
		if !ok {
			return nil, errors.New("no such whois server: " + host)
		}
		return net.Dial("tcp", ln.Addr().String())
	}
	t.Cleanup(func() { dialWhois = old })
}

func TestParseReferral(t *testing.T) {
	cases := map[string]string{
		"refer: whois.arin.net\n":                             "whois.arin.net",
		"whois:  whois.verisign-grs.com":                      "whois.verisign-grs.com",
		"Registrar WHOIS Server: whois.example-registrar.com": "whois.example-registrar.com",
		"ReferralServer: whois://whois.ripe.net:43":           "whois.ripe.net",
		"no referral here\ndomain: x":                         "",
	}
	for body, want := range cases {
		require.Equal(t, want, parseReferral(body), body)
	}
}

func TestQueryFollowsChain(t *testing.T) {
	fakeNet(t, map[string]*whoisServer{
		"whois.iana.org": {resp: "refer: registry.test\n"},
		"registry.test":  {resp: "domain: example.com\nRegistrar WHOIS Server: registrar.test\n"},
		"registrar.test": {resp: "domain: example.com\nRegistrar: Example Registrar\n"},
	})
	r, err := Query(context.Background(), "example.com", 3*time.Second)
	require.NoError(t, err)
	require.Contains(t, r.Raw, "Example Registrar")
	require.Equal(t, "registrar.test", r.Server)
	require.Equal(t, []string{"whois.iana.org", "registry.test", "registrar.test"}, r.Chain)
}

func TestQueryTerminal(t *testing.T) {
	fakeNet(t, map[string]*whoisServer{
		"whois.iana.org": {resp: "refer: registry.test\n"},
		"registry.test":  {resp: "domain: example.com\nno more referrals\n"},
	})
	r, err := Query(context.Background(), "example.com", 3*time.Second)
	require.NoError(t, err)
	require.Contains(t, r.Raw, "no more referrals")
	require.Equal(t, "registry.test", r.Server)
	require.Len(t, r.Chain, 2)
}

func TestQueryCapsHops(t *testing.T) {
	// Every server refers onward forever; the walk must stop at the cap.
	fakeNet(t, map[string]*whoisServer{
		"whois.iana.org": {resp: "refer: a.test\n"},
		"a.test":         {resp: "refer: b.test\n"},
		"b.test":         {resp: "refer: c.test\n"},
		"c.test":         {resp: "refer: d.test\n"},
		"d.test":         {resp: "record\n"},
	})
	r, err := Query(context.Background(), "example.com", 3*time.Second)
	require.NoError(t, err)
	require.LessOrEqual(t, len(r.Chain), 3) // iana + at most 2 hops
}

func TestQuerySendsObjectCRLF(t *testing.T) {
	iana := &whoisServer{resp: "no referral\n"}
	fakeNet(t, map[string]*whoisServer{"whois.iana.org": iana})
	_, err := Query(context.Background(), "1.1.1.1", 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "1.1.1.1\r\n", iana.gotQuery)
}

func TestQueryDialError(t *testing.T) {
	old := dialWhois
	dialWhois = func(context.Context, string) (net.Conn, error) { return nil, errors.New("boom") }
	t.Cleanup(func() { dialWhois = old })
	_, err := Query(context.Background(), "example.com", 2*time.Second)
	require.Error(t, err)
}

func TestQueryDomainUsesTLDForIANA(t *testing.T) {
	iana := &whoisServer{resp: "refer: registry.test\n"}
	fakeNet(t, map[string]*whoisServer{
		"whois.iana.org": iana,
		"registry.test":  {resp: "record\n"},
	})
	_, err := Query(context.Background(), "example.com", 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "com\r\n", iana.gotQuery) // IANA step queries the TLD
}
