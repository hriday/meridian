// Package sipping sends a SIP OPTIONS request — the canonical "ping" for SIP —
// to a PBX, trunk, or endpoint and reports whether it answers and what it
// supports. It's the first thing to check when VoIP signaling is in doubt.
// Point-in-time; nothing is stored.
package sipping

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// DialFunc establishes the transport connection; tests inject an in-memory one.
// For "tls" the caller dials TCP and wraps it.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// Result is the outcome of a SIP OPTIONS probe.
type Result struct {
	Target     string   `json:"target"`
	Transport  string   `json:"transport"`
	StatusCode int      `json:"statusCode"`
	Reason     string   `json:"reason"`
	Server     string   `json:"server,omitempty"`
	UserAgent  string   `json:"userAgent,omitempty"`
	Allow      []string `json:"allow,omitempty"`      // supported methods
	Accept     []string `json:"accept,omitempty"`     // supported body types
	Supported  []string `json:"supported,omitempty"`  // extensions
	RTTMillis  int      `json:"rttMillis"`
}

// Probe sends an OPTIONS to host:port over transport ("udp", "tcp", or "tls")
// and parses the response.
func Probe(ctx context.Context, dial DialFunc, host, port, transport string) (Result, error) {
	transport = strings.ToLower(transport)
	if transport == "" {
		transport = "udp"
	}
	network := transport
	if transport == "tls" {
		network = "tcp"
	}
	addr := net.JoinHostPort(host, port)

	start := time.Now()
	conn, err := dial(ctx, network, addr)
	if err != nil {
		return Result{}, fmt.Errorf("couldn't connect to %s/%s: %w", addr, transport, err)
	}
	defer conn.Close()
	if transport == "tls" {
		tconn := tls.Client(conn, &tls.Config{ServerName: host, InsecureSkipVerify: true}) //nolint:gosec // signaling probe
		if err := tconn.HandshakeContext(ctx); err != nil {
			return Result{}, fmt.Errorf("TLS handshake failed: %w", err)
		}
		conn = tconn
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	local := "127.0.0.1"
	if la, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		local = la.IP.String()
	} else if la, ok := conn.LocalAddr().(*net.TCPAddr); ok {
		local = la.IP.String()
	}

	req := buildOptions(host, port, transport, local)
	if _, err := conn.Write([]byte(req)); err != nil {
		return Result{}, fmt.Errorf("sending OPTIONS: %w", err)
	}

	resp, err := readResponse(conn)
	if err != nil {
		return Result{}, fmt.Errorf("reading response: %w", err)
	}
	res, err := parseResponse(resp)
	if err != nil {
		return Result{}, err
	}
	res.Target = addr
	res.Transport = transport
	res.RTTMillis = int(time.Since(start).Milliseconds())
	return res, nil
}

func buildOptions(host, port, transport, local string) string {
	branch := "z9hG4bK" + token()
	callID := token() + "@" + local
	tag := token()
	via := strings.ToUpper(transport)
	if transport == "tls" {
		via = "TLS"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "OPTIONS sip:%s:%s SIP/2.0\r\n", host, port)
	fmt.Fprintf(&b, "Via: SIP/2.0/%s %s:5060;branch=%s;rport\r\n", via, local, branch)
	b.WriteString("Max-Forwards: 70\r\n")
	fmt.Fprintf(&b, "From: <sip:meridian@%s>;tag=%s\r\n", local, tag)
	fmt.Fprintf(&b, "To: <sip:%s:%s>\r\n", host, port)
	fmt.Fprintf(&b, "Call-ID: %s\r\n", callID)
	b.WriteString("CSeq: 1 OPTIONS\r\n")
	fmt.Fprintf(&b, "Contact: <sip:meridian@%s:5060>\r\n", local)
	b.WriteString("User-Agent: Meridian\r\n")
	b.WriteString("Accept: application/sdp\r\n")
	b.WriteString("Content-Length: 0\r\n\r\n")
	return b.String()
}

// readResponse reads one SIP message: the status line + headers (ended by a
// blank line) plus any Content-Length body.
func readResponse(conn net.Conn) ([]byte, error) {
	br := bufio.NewReader(conn)
	var head []byte
	for {
		line, err := br.ReadBytes('\n')
		head = append(head, line...)
		if err != nil {
			if len(head) > 0 {
				return head, nil
			}
			return nil, err
		}
		if strings.TrimRight(string(line), "\r\n") == "" {
			break // end of headers
		}
	}
	n := contentLength(head)
	if n > 0 {
		body := make([]byte, n)
		_, _ = br.Read(body)
		head = append(head, body...)
	}
	return head, nil
}

func parseResponse(resp []byte) (Result, error) {
	lines := strings.Split(strings.ReplaceAll(string(resp), "\r\n", "\n"), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "SIP/2.0") {
		return Result{}, fmt.Errorf("not a SIP response (no SIP/2.0 status line)")
	}
	var res Result
	status := strings.Fields(lines[0])
	if len(status) >= 2 {
		res.StatusCode, _ = strconv.Atoi(status[1])
		res.Reason = strings.Join(status[2:], " ")
	}
	for _, ln := range lines[1:] {
		k, v, ok := strings.Cut(ln, ":")
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.TrimSpace(v)
		switch key {
		case "server":
			res.Server = val
		case "user-agent":
			res.UserAgent = val
		case "allow":
			res.Allow = splitList(val)
		case "accept":
			res.Accept = splitList(val)
		case "supported":
			res.Supported = splitList(val)
		}
	}
	return res, nil
}

func contentLength(head []byte) int {
	for _, ln := range strings.Split(string(head), "\n") {
		k, v, ok := strings.Cut(ln, ":")
		if ok && strings.EqualFold(strings.TrimSpace(k), "content-length") {
			n, _ := strconv.Atoi(strings.TrimSpace(v))
			return n
		}
	}
	return 0
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func token() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
