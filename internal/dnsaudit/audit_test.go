package dnsaudit

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

// auditServer options per test.
type serverOpts struct {
	nsServfail bool // NS query returns SERVFAIL
	dmarcNX    bool // _dmarc returns NXDOMAIN
	ad         bool // set the AuthenticatedData (DNSSEC) flag
	splitSPF   bool // apex SPF returned as a multi-string TXT split before -all
	nullMX     bool // MX is a null MX (0 .)
}

// auditServer starts a DNS server answering a healthy example domain, tweaked
// by opts, and returns its address.
func auditServer(t *testing.T, opts serverOpts) string {
	t.Helper()
	mux := mdns.NewServeMux()
	mux.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		m := new(mdns.Msg)
		m.SetReply(r)
		m.AuthenticatedData = opts.ad
		q := r.Question[0]
		name := q.Name
		switch {
		case q.Qtype == mdns.TypeNS:
			if opts.nsServfail {
				m.Rcode = mdns.RcodeServerFailure
				break
			}
			m.Answer = []mdns.RR{
				&mdns.NS{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeNS, Ttl: 3600}, Ns: "ns1.example.test."},
				&mdns.NS{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeNS, Ttl: 3600}, Ns: "ns2.example.test."},
			}
		case q.Qtype == mdns.TypeMX:
			if opts.nullMX {
				m.Answer = []mdns.RR{&mdns.MX{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeMX, Ttl: 3600}, Preference: 0, Mx: "."}}
				break
			}
			m.Answer = []mdns.RR{&mdns.MX{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeMX, Ttl: 3600}, Preference: 10, Mx: "mail.example.test."}}
		case q.Qtype == mdns.TypeTXT && strings.HasPrefix(name, "_dmarc."):
			if opts.dmarcNX {
				m.Rcode = mdns.RcodeNameError
				break
			}
			m.Answer = []mdns.RR{&mdns.TXT{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeTXT, Ttl: 3600}, Txt: []string{"v=DMARC1; p=reject; rua=mailto:dmarc@example.test"}}}
		case q.Qtype == mdns.TypeTXT:
			if opts.splitSPF {
				// A >255-byte SPF record is transmitted as multiple character-
				// strings; here the split falls right before the terminal -all.
				m.Answer = []mdns.RR{&mdns.TXT{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeTXT, Ttl: 3600}, Txt: []string{"v=spf1 include:_spf.example.com ", "-all"}}}
				break
			}
			m.Answer = []mdns.RR{&mdns.TXT{Hdr: mdns.RR_Header{Name: name, Rrtype: mdns.TypeTXT, Ttl: 3600}, Txt: []string{"v=spf1 -all"}}}
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

func checkByName(r Report, name string) (Check, bool) {
	for _, c := range r.Checks {
		if c.Name == name {
			return c, true
		}
	}
	return Check{}, false
}

func TestAuditHealthy(t *testing.T) {
	addr := auditServer(t, serverOpts{ad: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	for _, name := range []string{"Nameservers", "Mail (MX)", "SPF", "DMARC", "DNSSEC"} {
		c, ok := checkByName(r, name)
		require.True(t, ok, name)
		require.Equal(t, StatusOK, c.Status, name+": "+c.Note)
	}
	require.Equal(t, 0, r.Warns)
	require.Equal(t, 0, r.Missing)
	require.Equal(t, 0, r.Errors)
	require.Contains(t, strings.ToLower(r.Summary), "healthy")
}

func TestAuditMissingDMARC(t *testing.T) {
	addr := auditServer(t, serverOpts{ad: true, dmarcNX: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	c, ok := checkByName(r, "DMARC")
	require.True(t, ok)
	require.Equal(t, StatusMissing, c.Status)
	require.GreaterOrEqual(t, r.Missing, 1)
	require.Contains(t, r.Summary, "DMARC")
}

func TestAuditServfailIsError(t *testing.T) {
	addr := auditServer(t, serverOpts{nsServfail: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	c, ok := checkByName(r, "Nameservers")
	require.True(t, ok)
	require.Equal(t, StatusError, c.Status, "SERVFAIL must read as error, not missing")
	require.GreaterOrEqual(t, r.Errors, 1)
}

func TestAuditQuotedSPFParsed(t *testing.T) {
	// dnsx renders TXT values quoted; the SPF check must still parse it.
	addr := auditServer(t, serverOpts{ad: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	c, ok := checkByName(r, "SPF")
	require.True(t, ok)
	require.Equal(t, StatusOK, c.Status, "quoted v=spf1 must be parsed, not read as missing")
}

func TestAuditSplitSPFParsed(t *testing.T) {
	// A multi-string SPF record split before -all must still be graded ok (the
	// terminal mechanism is intact once chunks are joined), and the displayed
	// record must not carry embedded quotes.
	addr := auditServer(t, serverOpts{ad: true, splitSPF: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	c, ok := checkByName(r, "SPF")
	require.True(t, ok)
	require.Equal(t, StatusOK, c.Status, "split -all must read as strict, not warn: "+c.Note)
	require.NotEmpty(t, c.Records)
	require.NotContains(t, c.Records[0], `"`, "the record must be reconstructed without embedded quotes")
}

func TestAuditNullMX(t *testing.T) {
	addr := auditServer(t, serverOpts{ad: true, nullMX: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	c, ok := checkByName(r, "Mail (MX)")
	require.True(t, ok)
	require.Equal(t, StatusWarn, c.Status)
	require.Contains(t, strings.ToLower(c.Note), "no mail")
}

func TestAuditSummaryPrefersMissingOverWarn(t *testing.T) {
	// Null MX (warn, by design) + missing DMARC: the "fix first" headline must
	// name the genuinely-absent DMARC, not the deliberate null MX.
	addr := auditServer(t, serverOpts{ad: true, nullMX: true, dmarcNX: true})
	r, err := Audit(context.Background(), "example.test", addr, 3*time.Second)
	require.NoError(t, err)
	require.Contains(t, r.Summary, "DMARC")
	require.NotContains(t, r.Summary, "Mail (MX) needs attention")
}

func TestAuditRejectsEmpty(t *testing.T) {
	_, err := Audit(context.Background(), "", "127.0.0.1:53", 2*time.Second)
	require.Error(t, err)
	_, err = Audit(context.Background(), "example.test", "", 2*time.Second)
	require.Error(t, err)
}
