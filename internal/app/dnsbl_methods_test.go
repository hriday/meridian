package app

import (
	"net"
	"testing"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

// localDNSBL starts a DNS server that lists any queried name (A 127.0.0.2),
// returning its address, so Blocklist produces a populated report offline.
func localDNSBL(t *testing.T) string {
	t.Helper()
	mux := mdns.NewServeMux()
	mux.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		if r.Question[0].Qtype == mdns.TypeA {
			m.Answer = []mdns.RR{&mdns.A{
				Hdr: mdns.RR_Header{Name: r.Question[0].Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
				A:   net.ParseIP("127.0.0.2"),
			}}
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

func TestBlocklistMethod(t *testing.T) {
	addr := localDNSBL(t)
	rep, err := New().Blocklist("1.2.3.4", addr)
	require.NoError(t, err)
	require.Equal(t, addr, rep.Resolver)
	require.Greater(t, rep.Checked, 0)
	require.Equal(t, rep.Checked, rep.Listed) // every default zone lists on this stub
	require.NotEmpty(t, rep.Verdict)
}

func TestBlocklistMethodRejectsNonIP(t *testing.T) {
	_, err := New().Blocklist("example.com", "127.0.0.1:53")
	require.Error(t, err)
}
