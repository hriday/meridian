// Package axfr tests whether a domain's authoritative nameservers allow an
// unauthenticated DNS zone transfer (AXFR). A correctly configured server
// refuses AXFR from arbitrary clients; one that allows it hands a stranger the
// entire zone — every host, subdomain and internal record — in a single query,
// a classic information-disclosure finding. The probe is read-only: it asks for
// the transfer and reports whether the server would have given it.
package axfr

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	mdns "github.com/miekg/dns"

	"github.com/hriday/meridian/internal/dnsx"
)

// Status is a per-nameserver outcome.
type Status string

const (
	StatusRefused     Status = "refused"     // the server denied the transfer (secure)
	StatusTransferred Status = "transferred" // the server handed over the zone (EXPOSED)
	StatusError       Status = "error"       // the nameserver couldn't be reached/tested
)

// NSResult is one nameserver's outcome.
type NSResult struct {
	Nameserver string   `json:"nameserver"`
	Status     Status   `json:"status"`
	Records    int      `json:"records,omitempty"` // records returned if transferred
	Sample     []string `json:"sample,omitempty"`  // a few leaked names, if transferred
	Detail     string   `json:"detail,omitempty"`
}

// Report is the whole domain's AXFR exposure.
type Report struct {
	Domain      string     `json:"domain"`
	Nameservers []NSResult `json:"nameservers"`
	Exposed     bool       `json:"exposed"` // any nameserver allowed the transfer
	Summary     string     `json:"summary"`
}

// transferFn performs the actual AXFR against ns (host:port) for zone, returning
// the record count, a small sample of leaked names, and any error. Behind a var
// so tests can drive classify() without a live server.
var transferFn = doTransfer

// queryFn resolves DNS, behind a var so tests can inject nameservers.
var queryFn = dnsx.Query

// Test resolves domain's nameservers via resolver, then attempts a zone transfer
// against each and grades the result. It errors only if domain/resolver is empty
// or no nameservers can be found; an individual nameserver that can't be reached
// is a per-NS "error", not a failure of the whole probe.
func Test(ctx context.Context, domain, resolver string) (Report, error) {
	domain = strings.TrimSpace(strings.TrimSuffix(domain, "."))
	if domain == "" {
		return Report{}, errors.New("no domain to test")
	}
	if strings.TrimSpace(resolver) == "" {
		return Report{}, errors.New("no resolver specified")
	}
	nsRes, err := queryFn(ctx, domain, "NS", resolver)
	if err != nil {
		return Report{}, errors.New("couldn't look up nameservers: " + err.Error())
	}
	var servers []string
	for _, a := range nsRes.Answers {
		if a.Type == "NS" {
			servers = append(servers, strings.TrimSuffix(a.Value, "."))
		}
	}
	if len(servers) == 0 {
		return Report{}, errors.New("no nameservers found for " + domain)
	}

	rep := Report{Domain: domain}
	for _, ns := range servers {
		n, sample, terr := transferFn(ctx, domain, net.JoinHostPort(ns, "53"))
		rep.Nameservers = append(rep.Nameservers, classify(ns, n, sample, terr))
	}
	rep.Exposed = countExposed(rep.Nameservers) > 0
	rep.Summary = summarize(rep)
	return rep, nil
}

// classify turns one transfer attempt into a graded result. Any records returned
// means the zone was handed over, regardless of a trailing error. Otherwise a
// connection-level failure is "error" (we couldn't test), while a DNS-level
// denial is "refused" (the server correctly said no).
func classify(ns string, records int, sample []string, err error) NSResult {
	r := NSResult{Nameserver: ns}
	switch {
	case records > 0:
		r.Status, r.Records, r.Sample = StatusTransferred, records, sample
		r.Detail = "Zone transfer allowed — " + strconv.Itoa(records) + " records exposed to any client."
	case err != nil && isConnError(err):
		r.Status, r.Detail = StatusError, "Couldn't reach this nameserver over TCP/53: "+err.Error()
	default:
		r.Status, r.Detail = StatusRefused, "Zone transfer refused."
	}
	return r
}

// isConnError reports whether err is a network-level failure (dial/timeout),
// as opposed to a DNS-level refusal the server returned.
func isConnError(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	var oe *net.OpError
	if errors.As(err, &oe) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") || strings.Contains(s, "no such host") ||
		strings.Contains(s, "timeout") || strings.Contains(s, "unreachable") ||
		strings.Contains(s, "connection reset")
}

func countExposed(rs []NSResult) int {
	n := 0
	for _, r := range rs {
		if r.Status == StatusTransferred {
			n++
		}
	}
	return n
}

func summarize(r Report) string {
	exposed := countExposed(r.Nameservers)
	switch {
	case exposed > 0:
		return "EXPOSED — " + strconv.Itoa(exposed) + " of " + strconv.Itoa(len(r.Nameservers)) +
			" nameserver(s) allow a full zone transfer to anyone. Restrict AXFR to your secondaries."
	default:
		return "Secure — all " + strconv.Itoa(len(r.Nameservers)) + " nameserver(s) refused the zone transfer."
	}
}

// doTransfer runs the real AXFR, collecting records until the stream ends.
func doTransfer(ctx context.Context, zone, ns string) (int, []string, error) {
	tr := &mdns.Transfer{DialTimeout: 5 * time.Second, ReadTimeout: 8 * time.Second}
	m := new(mdns.Msg)
	m.SetAxfr(mdns.Fqdn(zone))
	ch, err := tr.In(m, ns)
	if err != nil {
		return 0, nil, err
	}
	var n int
	var sample []string
	for env := range ch {
		if env.Error != nil {
			return n, sample, env.Error
		}
		for _, rr := range env.RR {
			n++
			if len(sample) < 6 {
				h := rr.Header()
				sample = append(sample, strings.TrimSuffix(h.Name, ".")+" "+mdns.TypeToString[h.Rrtype])
			}
		}
	}
	return n, sample, nil
}
