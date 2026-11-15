// Package dnsx is a dig-like DNS query client over github.com/miekg/dns:
// it asks a chosen resolver for a chosen record type and reports the answer
// records, response flags, DNSSEC authentication, and round-trip time.
// Query does the network exchange; summarize is the pure result mapper.
package dnsx

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
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
	Transport     string   `json:"transport"` // udp, tcp, dot, doh
	Rcode         string   `json:"rcode"`
	Flags         []string `json:"flags"`
	Authenticated bool     `json:"authenticated"`
	RTTMillis     int      `json:"rttMillis"`
	Answers       []Record `json:"answers"`
}

// Query asks server (host or host:port; port 53 assumed) for name's rtype
// records over classic UDP (with a TCP retry on truncation), with EDNS0 and the
// DNSSEC-OK bit set. The deadline comes from ctx, else a 5s default.
func Query(ctx context.Context, name, rtype, server string) (Result, error) {
	return QueryVia(ctx, name, rtype, server, "udp")
}

// QueryVia is Query over a chosen transport: "udp" (default), "tcp", "dot"
// (DNS-over-TLS, port 853), or "doh" (DNS-over-HTTPS, RFC 8484). For dot/doh,
// server may be a hostname (verified) or IP; for doh it may also be a full URL.
func QueryVia(ctx context.Context, name, rtype, server, transport string) (Result, error) {
	m, err := buildQuery(name, rtype)
	if err != nil {
		return Result{}, err
	}
	timeout := 5 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.Until(dl)
	}

	var resp *mdns.Msg
	var rtt time.Duration
	switch strings.ToLower(strings.TrimSpace(transport)) {
	case "", "udp":
		resp, rtt, err = exchangeUDP(ctx, m, server, timeout)
	case "tcp":
		resp, rtt, err = (&mdns.Client{Net: "tcp", Timeout: timeout}).ExchangeContext(ctx, m, withPort(server, "53"))
	case "dot", "tls":
		resp, rtt, err = exchangeDoT(ctx, m, server, timeout)
	case "doh", "https":
		resp, rtt, err = exchangeDoH(ctx, m, server, timeout)
	default:
		return Result{}, fmt.Errorf("unknown transport %q (want udp, tcp, dot, or doh)", transport)
	}
	if err != nil {
		return Result{}, err
	}
	r := summarize(resp, server, rtt)
	r.Transport = normTransport(transport)
	return r, nil
}

func normTransport(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "", "udp":
		return "udp"
	case "tls", "dot":
		return "dot"
	case "https", "doh":
		return "doh"
	default:
		return strings.ToLower(t)
	}
}

func buildQuery(name, rtype string) (*mdns.Msg, error) {
	if rtype == "" {
		rtype = "A"
	}
	qtype, ok := mdns.StringToType[strings.ToUpper(rtype)]
	if !ok {
		return nil, fmt.Errorf("unknown record type %q", rtype)
	}
	m := new(mdns.Msg)
	m.SetQuestion(mdns.Fqdn(name), qtype)
	m.RecursionDesired = true
	m.SetEdns0(1232, true) // UDP payload 1232, DNSSEC OK
	return m, nil
}

// exchangeUDP does the classic query with a TCP retry on truncation/failure.
func exchangeUDP(ctx context.Context, m *mdns.Msg, server string, timeout time.Duration) (*mdns.Msg, time.Duration, error) {
	addr := withPort(server, "53")
	c := &mdns.Client{Timeout: timeout}
	resp, rtt, err := c.ExchangeContext(ctx, m, addr)
	// Resolvers truncate large responses and expect a TCP retry; an oversized UDP
	// datagram also surfaces as a parse error. TCP has no size limit.
	if err != nil || (resp != nil && resp.Truncated) {
		tcp := &mdns.Client{Net: "tcp", Timeout: timeout}
		if r2, rtt2, err2 := tcp.ExchangeContext(ctx, m, addr); err2 == nil {
			return r2, rtt2, nil
		} else if err == nil {
			err = err2
		}
	}
	return resp, rtt, err
}

// exchangeDoT does DNS-over-TLS (port 853). A hostname server is name-verified;
// an IP can't be, so the TLS name check is skipped for IPs.
func exchangeDoT(ctx context.Context, m *mdns.Msg, server string, timeout time.Duration) (*mdns.Msg, time.Duration, error) {
	addr := withPort(server, "853")
	host, _, _ := net.SplitHostPort(addr)
	c := &mdns.Client{Net: "tcp-tls", Timeout: timeout}
	if net.ParseIP(host) != nil {
		c.TLSConfig = &tls.Config{InsecureSkipVerify: true} // IP target: no name to verify
	} else {
		c.TLSConfig = &tls.Config{ServerName: host}
	}
	return c.ExchangeContext(ctx, m, addr)
}

// exchangeDoH does DNS-over-HTTPS (RFC 8484): POST the wire-format query as
// application/dns-message. server may be a URL or a host (→ https://host/dns-query).
func exchangeDoH(ctx context.Context, m *mdns.Msg, server string, timeout time.Duration) (*mdns.Msg, time.Duration, error) {
	url := server
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + strings.TrimSuffix(server, "/") + "/dns-query"
	}
	packed, err := m.Pack()
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(packed))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	t0 := time.Now()
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("DoH HTTP %d from %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, 0, err
	}
	rtt := time.Since(t0)
	out := new(mdns.Msg)
	if err := out.Unpack(body); err != nil {
		return nil, 0, fmt.Errorf("parsing DoH response: %w", err)
	}
	return out, rtt, nil
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
