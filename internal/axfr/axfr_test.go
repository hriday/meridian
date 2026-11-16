package axfr

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"

	"github.com/hriday/meridian/internal/dnsx"
)

func TestClassify(t *testing.T) {
	// Records returned => transferred (exposed), even with a trailing error.
	r := classify("ns1.", 42, []string{"a.example. A"}, errors.New("stream ended"))
	require.Equal(t, StatusTransferred, r.Status)
	require.Equal(t, 42, r.Records)

	// A DNS-level refusal (an rcode response) => refused/secure.
	r = classify("ns2.", 0, nil, errors.New("dns: bad xfr rcode: 5"))
	require.Equal(t, StatusRefused, r.Status)

	// A connection-level failure (net type) => couldn't test, never "secure".
	r = classify("ns3.", 0, nil, &net.OpError{Op: "dial", Err: errors.New("connection refused")})
	require.Equal(t, StatusError, r.Status)

	// A transport failure that isn't a net.* type (e.g. a wrapped syscall string)
	// must still be "error", not fall through to "refused".
	r = classify("ns3b.", 0, nil, errors.New("write tcp 1.2.3.4:53: broken pipe"))
	require.Equal(t, StatusError, r.Status, "a broken-pipe transport error is not a refusal")

	// No records, no error => connected but served nothing; refused, never exposed.
	r = classify("ns4.", 0, nil, nil)
	require.Equal(t, StatusRefused, r.Status)
}

func TestTestOrchestration(t *testing.T) {
	defer func(q func(context.Context, string, string, string) (dnsx.Result, error)) { queryFn = q }(queryFn)
	defer func(tr func(context.Context, string, string) (int, []string, error)) { transferFn = tr }(transferFn)

	queryFn = func(_ context.Context, _, _, _ string) (dnsx.Result, error) {
		return dnsx.Result{Answers: []dnsx.Record{
			{Type: "NS", Value: "open.ns.test."},
			{Type: "NS", Value: "closed.ns.test."},
		}}, nil
	}
	transferFn = func(_ context.Context, _, ns string) (int, []string, error) {
		if ns == "open.ns.test:53" {
			return 7, []string{"vpn.example A", "db.example A"}, nil
		}
		return 0, nil, errors.New("dns: bad xfr rcode: 5")
	}
	rep, err := Test(context.Background(), "example.com", "1.1.1.1")
	require.NoError(t, err)
	require.True(t, rep.Exposed, "one NS allowed the transfer")
	require.Len(t, rep.Nameservers, 2)
	require.Contains(t, rep.Summary, "EXPOSED")
}

// TestDoTransferAgainstLiveServers exercises the real miekg transfer path: one
// server that serves AXFR, one that refuses it.
func TestDoTransferAgainstLiveServers(t *testing.T) {
	allow := startAXFRServer(t, true)
	refuse := startAXFRServer(t, false)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	n, sample, err := doTransfer(ctx, "example.test", allow)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 2, "the allowing server hands over the zone")
	require.NotEmpty(t, sample)

	n, _, err = doTransfer(ctx, "example.test", refuse)
	require.Equal(t, 0, n, "the refusing server returns no records")
	require.Error(t, err, "a refusal surfaces as an error with zero records")
	require.False(t, countExposed([]NSResult{classify("x", n, nil, err)}) > 0)
}

// startAXFRServer starts a TCP DNS server for example.test. that either serves
// or refuses AXFR, and returns its host:port.
func startAXFRServer(t *testing.T, allow bool) string {
	t.Helper()
	mux := mdns.NewServeMux()
	mux.HandleFunc("example.test.", func(w mdns.ResponseWriter, r *mdns.Msg) {
		if r.Question[0].Qtype != mdns.TypeAXFR {
			m := new(mdns.Msg)
			m.SetReply(r)
			_ = w.WriteMsg(m)
			return
		}
		if !allow {
			m := new(mdns.Msg)
			m.SetRcode(r, mdns.RcodeRefused)
			_ = w.WriteMsg(m)
			return
		}
		soa := &mdns.SOA{Hdr: mdns.RR_Header{Name: "example.test.", Rrtype: mdns.TypeSOA, Class: mdns.ClassINET, Ttl: 3600}, Ns: "ns1.example.test.", Mbox: "hostmaster.example.test.", Serial: 1}
		a := &mdns.A{Hdr: mdns.RR_Header{Name: "vpn.example.test.", Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 3600}, A: net.ParseIP("10.0.0.1")}
		tr := new(mdns.Transfer)
		ch := make(chan *mdns.Envelope)
		go func() {
			ch <- &mdns.Envelope{RR: []mdns.RR{soa, a, soa}}
			close(ch)
		}()
		_ = tr.Out(w, r, ch)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &mdns.Server{Listener: ln, Handler: mux}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })
	return ln.Addr().String()
}
