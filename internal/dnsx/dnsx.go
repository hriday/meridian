// Package dnsx is a dig-like DNS query client over github.com/miekg/dns:
// it asks a chosen resolver for a chosen record type and reports the answer
// records, response flags, DNSSEC authentication, and round-trip time.
// Query does the network exchange; summarize is the pure result mapper.
package dnsx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	mdns "github.com/miekg/dns"
	"github.com/quic-go/quic-go"
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

// JoinTXT turns a TXT answer Value — which this package formats as
// space-separated quoted character-strings (`"part1" "part2"`) — into the single
// string DNS defines it to be: the character-strings' contents concatenated with
// no added separator, so a long SPF/DKIM record split across strings reassembles
// correctly. A value with no quotes is returned trimmed.
func JoinTXT(value string) string {
	if !strings.Contains(value, "\"") {
		return strings.TrimSpace(value)
	}
	var b strings.Builder
	inQuote, esc := false, false
	for _, r := range value {
		switch {
		case esc:
			b.WriteRune(r)
			esc = false
		case r == '\\':
			esc = true
		case r == '"':
			inQuote = !inQuote
		case inQuote:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Query asks server (host or host:port; port 53 assumed) for name's rtype
// records over classic UDP (with a TCP retry on truncation), with EDNS0 and the
// DNSSEC-OK bit set. The deadline comes from ctx, else a 5s default.
func Query(ctx context.Context, name, rtype, server string) (Result, error) {
	return QueryVia(ctx, name, rtype, server, "udp")
}

// QueryMsg is like Query but returns the raw DNS message, so callers that need
// the actual resource records — not their string form — can reach them (e.g.
// DNSSEC signature verification needs the DNSKEY/DS/RRSIG RRs themselves). UDP
// with the same TCP-retry-on-truncation as Query; the DNSSEC-OK bit is set.
func QueryMsg(ctx context.Context, name, rtype, server string) (*mdns.Msg, error) {
	m, err := buildQuery(name, rtype)
	if err != nil {
		return nil, err
	}
	timeout := 5 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		timeout = time.Until(dl)
	}
	resp, _, err := exchangeUDP(ctx, m, server, timeout)
	return resp, err
}

// QueryVia is Query over a chosen transport: "udp" (default), "tcp", "dot"
// (DNS-over-TLS, port 853), "doh" (DNS-over-HTTPS, RFC 8484), or "doq"
// (DNS-over-QUIC, RFC 9250, port 853). For dot/doq, server may be a hostname
// (verified) or IP; for doh it may also be a full URL.
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
	case "doq", "quic":
		resp, rtt, err = exchangeDoQ(ctx, m, server, timeout)
	default:
		return Result{}, fmt.Errorf("unknown transport %q (want udp, tcp, dot, doh, or doq)", transport)
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
	case "quic", "doq":
		return "doq"
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

// doqEncode frames a DNS message for DoQ (RFC 9250): the message ID is set to 0
// (the QUIC stream provides query/response correlation) and the wire message is
// 2-byte length-prefixed, as in DNS-over-TCP.
func doqEncode(m *mdns.Msg) ([]byte, error) {
	m.Id = 0
	packed, err := m.Pack()
	if err != nil {
		return nil, err
	}
	if len(packed) > 0xffff {
		return nil, fmt.Errorf("DoQ query too large: %d bytes", len(packed))
	}
	buf := make([]byte, 2+len(packed))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(packed)))
	copy(buf[2:], packed)
	return buf, nil
}

// doqDecode parses a 2-byte length-prefixed DoQ response.
func doqDecode(b []byte) (*mdns.Msg, error) {
	if len(b) < 2 {
		return nil, fmt.Errorf("short DoQ response (%d bytes)", len(b))
	}
	n := int(binary.BigEndian.Uint16(b[:2]))
	if len(b)-2 < n {
		return nil, fmt.Errorf("truncated DoQ response: have %d, want %d", len(b)-2, n)
	}
	out := new(mdns.Msg)
	if err := out.Unpack(b[2 : 2+n]); err != nil {
		return nil, fmt.Errorf("parsing DoQ response: %w", err)
	}
	return out, nil
}

// exchangeDoQ does DNS-over-QUIC (RFC 9250, port 853, ALPN "doq"): one query per
// bidirectional stream, each message 2-byte length-prefixed. A hostname server
// is name-verified; an IP can't be, so the TLS name check is skipped for IPs.
func exchangeDoQ(ctx context.Context, m *mdns.Msg, server string, timeout time.Duration) (*mdns.Msg, time.Duration, error) {
	addr := withPort(server, "853")
	host, _, _ := net.SplitHostPort(addr)
	tlsConf := &tls.Config{NextProtos: []string{"doq"}}
	if net.ParseIP(host) != nil {
		tlsConf.InsecureSkipVerify = true // IP target: no name to verify
	} else {
		tlsConf.ServerName = host
	}
	query, err := doqEncode(m)
	if err != nil {
		return nil, 0, err
	}

	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t0 := time.Now()
	conn, err := quic.DialAddr(dctx, addr, tlsConf, &quic.Config{HandshakeIdleTimeout: timeout})
	if err != nil {
		return nil, 0, fmt.Errorf("DoQ dial: %w", err)
	}
	defer conn.CloseWithError(0, "")

	stream, err := conn.OpenStreamSync(dctx)
	if err != nil {
		return nil, 0, err
	}
	if _, err := stream.Write(query); err != nil {
		return nil, 0, err
	}
	stream.Close() // half-close: the client's query is complete

	resp, err := io.ReadAll(io.LimitReader(stream, 0xffff+2))
	if err != nil && len(resp) == 0 {
		return nil, 0, err
	}
	out, err := doqDecode(resp)
	if err != nil {
		return nil, 0, err
	}
	return out, time.Since(t0), nil
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
