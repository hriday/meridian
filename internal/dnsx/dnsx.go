// Package dnsx is a dig-like DNS query client over github.com/miekg/dns:
// it asks a chosen resolver for a chosen record type and reports the answer
// records, response flags, DNSSEC authentication, and round-trip time.
// Query does the network exchange; summarize is the pure result mapper.
package dnsx

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
)

// Record is one answer resource record in text form.
type Record struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	TTL   uint32 `json:"ttl"`
	Value string `json:"value"`
}

// Result is the outcome of one DNS query.
type Result struct {
	Server        string   `json:"server"`
	Rcode         string   `json:"rcode"`
	Flags         []string `json:"flags"`
	Authenticated bool     `json:"authenticated"`
	RTTMillis     int      `json:"rttMillis"`
	Answers       []Record `json:"answers"`
}

// Query asks server (host or host:port; port 53 assumed) for name's rtype
// records (A, AAAA, MX, NS, TXT, CNAME, SOA, PTR, CAA, ANY, …), with EDNS0 and
// the DNSSEC-OK bit set. The deadline comes from ctx, else a 5s default.
func Query(ctx context.Context, name, rtype, server string) (Result, error) {
	if rtype == "" {
		rtype = "A"
	}
	qtype, ok := mdns.StringToType[strings.ToUpper(rtype)]
	if !ok {
		return Result{}, fmt.Errorf("unknown record type %q", rtype)
	}

	m := new(mdns.Msg)
	m.SetQuestion(mdns.Fqdn(name), qtype)
	m.RecursionDesired = true
	m.SetEdns0(1232, true) // UDP payload 1232, DNSSEC OK

	c := new(mdns.Client)
	c.Timeout = 5 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		c.Timeout = time.Until(dl)
	}

	addr := withPort(server, "53")
	resp, rtt, err := c.ExchangeContext(ctx, m, addr)
	// Fall back to TCP when UDP fails or the answer is truncated. Real resolvers
	// truncate large responses (e.g. a domain's apex TXT) and expect a TCP retry;
	// an oversized UDP datagram also surfaces here as a parse error. TCP has no
	// size limit, so the retry gets the full answer.
	if err != nil || (resp != nil && resp.Truncated) {
		tcp := &mdns.Client{Net: "tcp", Timeout: c.Timeout}
		if r2, rtt2, err2 := tcp.ExchangeContext(ctx, m, addr); err2 == nil {
			resp, rtt, err = r2, rtt2, nil
		} else if err == nil {
			err = err2
		}
	}
	if err != nil {
		return Result{}, err
	}
	return summarize(resp, server, rtt), nil
}

// summarize maps a DNS response message into a Result.
func summarize(msg *mdns.Msg, server string, rtt time.Duration) Result {
	res := Result{
		Server:        server,
		Rcode:         mdns.RcodeToString[msg.Rcode],
		Authenticated: msg.AuthenticatedData,
		RTTMillis:     int(rtt.Milliseconds()),
	}
	for flag, on := range map[string]bool{
		"qr": msg.Response, "aa": msg.Authoritative, "tc": msg.Truncated,
		"rd": msg.RecursionDesired, "ra": msg.RecursionAvailable,
		"ad": msg.AuthenticatedData, "cd": msg.CheckingDisabled,
	} {
		if on {
			res.Flags = append(res.Flags, flag)
		}
	}
	for _, rr := range msg.Answer {
		h := rr.Header()
		value := strings.TrimSpace(strings.TrimPrefix(rr.String(), h.String()))
		res.Answers = append(res.Answers, Record{
			Name:  h.Name,
			Type:  mdns.TypeToString[h.Rrtype],
			TTL:   h.Ttl,
			Value: value,
		})
	}
	return res
}

func withPort(server, port string) string {
	if _, _, err := net.SplitHostPort(server); err == nil {
		return server
	}
	return net.JoinHostPort(server, port)
}
