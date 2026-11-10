package dnsbl

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

// dnsblServer starts a local DNS server that answers A/TXT queries by zone
// suffix: a name under ".listed.test." is listed (127.0.0.2 + a TXT reason), one
// under ".blocked.test." returns the public-resolver error code, and anything
// else is NXDOMAIN (clean). Returns the server address (host:port).
func dnsblServer(t *testing.T) string {
	t.Helper()
	mux := mdns.NewServeMux()
	mux.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		q := r.Question[0]
		name := q.Name
		switch {
		case strings.HasSuffix(name, ".listed.test."):
			if q.Qtype == mdns.TypeA {
				m.Answer = []mdns.RR{&mdns.A{
					Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
					A:   net.ParseIP("127.0.0.2"),
				}}
			} else if q.Qtype == mdns.TypeTXT {
				m.Answer = []mdns.RR{&mdns.TXT{
					Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeTXT, Class: mdns.ClassINET, Ttl: 60},
					Txt: []string{"listed: see https://example.com/remove"},
				}}
			}
		case strings.HasSuffix(name, ".blocked.test."):
			if q.Qtype == mdns.TypeA {
				m.Answer = []mdns.RR{&mdns.A{
					Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
					A:   net.ParseIP("127.255.255.254"),
				}}
			}
		default:
			m.Rcode = mdns.RcodeNameError // NXDOMAIN = clean
		}
		_ = w.WriteMsg(m)
	})
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &mdns.Server{PacketConn: pc, Handler: mux}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })
	return pc.LocalAddr().String()
}

func findListing(r Report, zoneHost string) (Listing, bool) {
	for _, l := range r.Listings {
		if l.Zone == zoneHost {
			return l, true
		}
	}
	return Listing{}, false
}

func TestCheckListedAndClean(t *testing.T) {
	addr := dnsblServer(t)
	zones := []Zone{
		{Host: "listed.test", Name: "Listed List"},
		{Host: "clean.test", Name: "Clean List"},
	}
	r, err := Check(context.Background(), "1.2.3.4", addr, zones, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, 1, r.Listed)
	require.Equal(t, 2, r.Checked)
	require.Equal(t, 0, r.Unreachable)

	l, ok := findListing(r, "listed.test")
	require.True(t, ok)
	require.Equal(t, StatusListed, l.Status)
	require.Contains(t, l.Codes, "127.0.0.2")
	require.Contains(t, l.Reason, "example.com")

	c, ok := findListing(r, "clean.test")
	require.True(t, ok)
	require.Equal(t, StatusClean, c.Status)

	require.True(t, strings.HasPrefix(r.Verdict, "Listed on 1"), r.Verdict)
}

func TestCheckAllClean(t *testing.T) {
	addr := dnsblServer(t)
	zones := []Zone{
		{Host: "clean.test", Name: "Clean A"},
		{Host: "clean2.test", Name: "Clean B"},
	}
	r, err := Check(context.Background(), "9.9.9.9", addr, zones, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, 0, r.Listed)
	require.Equal(t, 0, r.Unreachable)
	require.Equal(t, "Not listed", r.Verdict)
	require.Contains(t, r.Summary, "2")
}

func TestCheckBlockedIncomplete(t *testing.T) {
	addr := dnsblServer(t)
	zones := []Zone{
		{Host: "blocked.test", Name: "Blocked List"},
		{Host: "clean.test", Name: "Clean List"},
	}
	r, err := Check(context.Background(), "1.2.3.4", addr, zones, 3*time.Second)
	require.NoError(t, err)

	b, ok := findListing(r, "blocked.test")
	require.True(t, ok)
	require.Equal(t, StatusBlocked, b.Status)

	require.GreaterOrEqual(t, r.Unreachable, 1)
	require.Equal(t, 0, r.Listed) // a blocked query is not a listing
	require.Contains(t, strings.ToLower(r.Summary), "incomplete")
}

func TestCheckRejectsNonIP(t *testing.T) {
	_, err := Check(context.Background(), "example.com", "127.0.0.1:53", DefaultZones(), 2*time.Second)
	require.Error(t, err)
}

func TestCheckRejectsEmptyResolver(t *testing.T) {
	_, err := Check(context.Background(), "1.2.3.4", "", DefaultZones(), 2*time.Second)
	require.Error(t, err)
}
