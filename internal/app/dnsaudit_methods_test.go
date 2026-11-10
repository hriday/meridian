package app

import (
	"net"
	"testing"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

// localDNSAuditServer answers a healthy domain so DNSAudit produces a populated
// report offline.
func localDNSAuditServer(t *testing.T) string {
	t.Helper()
	mux := mdns.NewServeMux()
	mux.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		m.AuthenticatedData = true
		q := r.Question[0]
		switch {
		case q.Qtype == mdns.TypeNS:
			m.Answer = []mdns.RR{&mdns.NS{Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeNS, Ttl: 3600}, Ns: "ns1.example.test."}}
		case q.Qtype == mdns.TypeMX:
			m.Answer = []mdns.RR{&mdns.MX{Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeMX, Ttl: 3600}, Preference: 10, Mx: "mail.example.test."}}
		case q.Qtype == mdns.TypeTXT && len(q.Name) > 7 && q.Name[:7] == "_dmarc.":
			m.Answer = []mdns.RR{&mdns.TXT{Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeTXT, Ttl: 3600}, Txt: []string{"v=DMARC1; p=reject"}}}
		case q.Qtype == mdns.TypeTXT:
			m.Answer = []mdns.RR{&mdns.TXT{Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeTXT, Ttl: 3600}, Txt: []string{"v=spf1 -all"}}}
		default:
			m.Rcode = mdns.RcodeNameError
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

func TestDNSAuditMethod(t *testing.T) {
	addr := localDNSAuditServer(t)
	rep, err := New().DNSAudit("example.test", addr)
	require.NoError(t, err)
	require.Equal(t, addr, rep.Resolver)
	require.Len(t, rep.Checks, 5)
	require.NotEmpty(t, rep.Summary)
}

func TestDNSAuditMethodRejectsEmpty(t *testing.T) {
	addr := localDNSAuditServer(t)
	_, err := New().DNSAudit("", addr)
	require.Error(t, err)
}
