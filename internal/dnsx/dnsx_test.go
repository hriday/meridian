package dnsx

import (
	"context"
	"net"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func aRR(name, ip string, ttl uint32) mdns.RR {
	return &mdns.A{
		Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: ttl},
		A:   net.ParseIP(ip),
	}
}

func TestSummarizeAAndFlags(t *testing.T) {
	m := new(mdns.Msg)
	m.Rcode = mdns.RcodeSuccess
	m.Response = true
	m.RecursionDesired = true
	m.RecursionAvailable = true
	m.AuthenticatedData = true
	m.Answer = []mdns.RR{aRR("cloudflare.com.", "104.16.132.229", 300), aRR("cloudflare.com.", "104.16.133.229", 300)}

	r := summarize(m, "192.168.1.1", 11*time.Millisecond)
	require.Equal(t, "NOERROR", r.Rcode)
	require.True(t, r.Authenticated)
	require.Len(t, r.Answers, 2)
	require.Equal(t, "A", r.Answers[0].Type)
	require.Equal(t, "104.16.132.229", r.Answers[0].Value)
	require.Equal(t, uint32(300), r.Answers[0].TTL)
	require.Subset(t, r.Flags, []string{"qr", "rd", "ra", "ad"})
}

func TestSummarizeNXDOMAIN(t *testing.T) {
	m := new(mdns.Msg)
	m.Rcode = mdns.RcodeNameError
	m.Response = true
	r := summarize(m, "1.1.1.1", 4*time.Millisecond)
	require.Equal(t, "NXDOMAIN", r.Rcode)
	require.Empty(t, r.Answers)
}

func TestSummarizeRecordTypes(t *testing.T) {
	m := new(mdns.Msg)
	m.Rcode = mdns.RcodeSuccess
	m.Answer = []mdns.RR{
		&mdns.MX{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeMX, Ttl: 3600}, Preference: 10, Mx: "mail.example.com."},
		&mdns.TXT{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeTXT, Ttl: 3600}, Txt: []string{"v=spf1 -all"}},
		&mdns.CAA{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeCAA, Ttl: 3600}, Flag: 0, Tag: "issue", Value: "letsencrypt.org"},
		&mdns.AAAA{Hdr: mdns.RR_Header{Name: "example.com.", Rrtype: mdns.TypeAAAA, Ttl: 300}, AAAA: net.ParseIP("2606:4700::6810:84e5")},
	}
	r := summarize(m, "8.8.8.8", time.Millisecond)
	require.Len(t, r.Answers, 4)
	require.Equal(t, "MX", r.Answers[0].Type)
	require.Contains(t, r.Answers[0].Value, "mail.example.com.")
	require.Equal(t, "TXT", r.Answers[1].Type)
	require.Contains(t, r.Answers[1].Value, "v=spf1 -all")
	require.Equal(t, "CAA", r.Answers[2].Type)
	require.Contains(t, r.Answers[2].Value, "issue")
	require.Equal(t, "AAAA", r.Answers[3].Type)
	require.Contains(t, r.Answers[3].Value, "2606:4700")
}

func TestQueryAgainstLocalServer(t *testing.T) {
	mux := mdns.NewServeMux()
	mux.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		m.Answer = []mdns.RR{aRR(r.Question[0].Name, "192.0.2.7", 60)}
		_ = w.WriteMsg(m)
	})
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &mdns.Server{PacketConn: pc, Handler: mux}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := Query(ctx, "test.example.", "A", pc.LocalAddr().String())
	require.NoError(t, err)
	require.Equal(t, "NOERROR", res.Rcode)
	require.Len(t, res.Answers, 1)
	require.Equal(t, "192.0.2.7", res.Answers[0].Value)
}

// TestQueryFallsBackToTCPOnTruncation: a resolver that truncates the UDP answer
// (TC bit, no records) but serves the full answer over TCP. Query must retry
// over TCP and return the full answer — real resolvers truncate large responses
// (e.g. a domain's apex TXT) exactly this way.
func TestQueryFallsBackToTCPOnTruncation(t *testing.T) {
	udpHandler := mdns.NewServeMux()
	udpHandler.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		m.Truncated = true // "too big for UDP — ask over TCP"
		_ = w.WriteMsg(m)
	})
	tcpHandler := mdns.NewServeMux()
	tcpHandler.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		m.Answer = []mdns.RR{aRR(r.Question[0].Name, "192.0.2.9", 60)}
		_ = w.WriteMsg(m)
	})
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := pc.LocalAddr().String()
	udpSrv := &mdns.Server{PacketConn: pc, Handler: udpHandler}
	go func() { _ = udpSrv.ActivateAndServe() }()
	t.Cleanup(func() { _ = udpSrv.Shutdown() })

	ln, err := net.Listen("tcp", addr) // same host:port, TCP
	require.NoError(t, err)
	tcpSrv := &mdns.Server{Listener: ln, Handler: tcpHandler}
	go func() { _ = tcpSrv.ActivateAndServe() }()
	t.Cleanup(func() { _ = tcpSrv.Shutdown() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := Query(ctx, "big.example.", "A", addr)
	require.NoError(t, err)
	require.Len(t, res.Answers, 1)
	require.Equal(t, "192.0.2.9", res.Answers[0].Value)
}

// TestQueryFallsBackToTCPOnUDPError: a TCP-only resolver (no UDP listener on the
// port). The UDP exchange fails; Query must fall back to TCP and still answer.
// This is the path that fixes the live "overflowing header size" UDP error.
func TestQueryFallsBackToTCPOnUDPError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	tcpHandler := mdns.NewServeMux()
	tcpHandler.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		m.Answer = []mdns.RR{aRR(r.Question[0].Name, "192.0.2.11", 60)}
		_ = w.WriteMsg(m)
	})
	tcpSrv := &mdns.Server{Listener: ln, Handler: tcpHandler}
	go func() { _ = tcpSrv.ActivateAndServe() }()
	t.Cleanup(func() { _ = tcpSrv.Shutdown() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := Query(ctx, "tcponly.example.", "A", addr)
	require.NoError(t, err)
	require.Len(t, res.Answers, 1)
	require.Equal(t, "192.0.2.11", res.Answers[0].Value)
}
