// Package smtpprobe opens an SMTP conversation with a mail server and reports
// what it finds: the banner, the EHLO capabilities, whether STARTTLS is offered,
// and — after upgrading (or on an implicit-TLS port) — the server certificate.
// It complements the email-auth tool (SPF/DKIM/DMARC) by actually talking to
// the MX instead of only reading DNS.
//
// It never authenticates, never sends mail, and never tests open relay; it only
// reads what the server advertises. Point-in-time; nothing stored.
package smtpprobe

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

// DialFunc establishes the raw TCP connection; net.Dialer.DialContext fits, and
// tests inject an in-memory server.
type DialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// TopMX returns the highest-priority (lowest-preference) MX host for a domain,
// so "probe example.com" can reach whatever actually receives its mail.
func TopMX(ctx context.Context, domain string) (string, error) {
	mxs, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err != nil {
		// A failed lookup is not the same as "no MX" — don't word a resolver
		// error as a definitive absence.
		return "", fmt.Errorf("couldn't look up MX for %s: %w", domain, err)
	}
	if len(mxs) == 0 {
		return "", fmt.Errorf("no MX records for %s", domain)
	}
	best := mxs[0]
	for _, m := range mxs[1:] {
		if m.Pref < best.Pref {
			best = m
		}
	}
	return strings.TrimSuffix(best.Host, "."), nil
}

// TLSInfo describes the server certificate seen after TLS is established.
type TLSInfo struct {
	Version   string    `json:"version"`
	Cipher    string    `json:"cipher"`
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	DNSNames  []string  `json:"dnsNames"`
	NotAfter  time.Time `json:"notAfter"`
	DaysLeft  int       `json:"daysLeft"`
	HostMatch bool      `json:"hostMatch"` // cert is valid for the probed host
}

// Result is the outcome of one probe.
type Result struct {
	Host         string   `json:"host"`
	Port         string   `json:"port"`
	Banner       string   `json:"banner"`
	Greeting     string   `json:"greeting"` // first line of the EHLO 250 reply
	Capabilities []string `json:"capabilities"`
	StartTLS     bool     `json:"startTLS"`      // STARTTLS advertised
	TLS          *TLSInfo `json:"tls,omitempty"` // present if TLS was established
	ImplicitTLS  bool     `json:"implicitTLS"`   // connected with TLS from the start (e.g. :465)
	AuthMechs    []string `json:"authMechs,omitempty"`
	RTTMillis    int      `json:"rttMillis"`
}

// Probe connects to host:port, runs EHLO, and (unless implicit TLS) upgrades via
// STARTTLS when offered, capturing the certificate. ehlo is the name sent in the
// EHLO command.
func Probe(ctx context.Context, dial DialFunc, host, port, ehlo string, implicitTLS bool) (Result, error) {
	if ehlo == "" {
		ehlo = "meridian.probe"
	}
	addr := net.JoinHostPort(host, port)
	start := time.Now()
	conn, err := dial(ctx, "tcp", addr)
	if err != nil {
		return Result{}, fmt.Errorf("couldn't connect to %s: %w", addr, err)
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	res := Result{Host: host, Port: port, ImplicitTLS: implicitTLS}
	tlsConf := &tls.Config{ServerName: host, InsecureSkipVerify: true} //nolint:gosec // we inspect, not trust

	if implicitTLS {
		tconn := tls.Client(conn, tlsConf)
		if err := tconn.HandshakeContext(ctx); err != nil {
			return Result{}, fmt.Errorf("TLS handshake failed: %w", err)
		}
		res.TLS = tlsInfo(tconn, host)
		conn = tconn
	}

	br := bufio.NewReader(conn)
	banner, err := readReply(br)
	if err != nil {
		return Result{}, fmt.Errorf("reading banner: %w", err)
	}
	res.Banner = firstLine(banner)

	caps, greet, err := ehloExchange(conn, br, ehlo)
	if err != nil {
		return Result{}, err
	}
	res.Greeting = greet
	res.Capabilities = caps
	res.StartTLS = hasCap(caps, "STARTTLS")
	res.AuthMechs = authMechs(caps)

	if !implicitTLS && res.StartTLS {
		if _, err := fmt.Fprint(conn, "STARTTLS\r\n"); err == nil {
			if reply, err := readReply(br); err == nil && strings.HasPrefix(reply, "220") {
				tconn := tls.Client(conn, tlsConf)
				if err := tconn.HandshakeContext(ctx); err == nil {
					res.TLS = tlsInfo(tconn, host)
					conn = tconn
					// re-EHLO over TLS to refresh capabilities (best-effort)
					br = bufio.NewReader(conn)
					if caps2, greet2, err := ehloExchange(conn, br, ehlo); err == nil {
						res.Capabilities = caps2
						res.Greeting = greet2
						res.AuthMechs = authMechs(caps2)
					}
				}
			}
		}
	}

	fmt.Fprint(conn, "QUIT\r\n")
	res.RTTMillis = int(time.Since(start).Milliseconds())
	return res, nil
}

// ehloExchange sends EHLO and parses the multiline 250 reply into capabilities.
func ehloExchange(conn net.Conn, br *bufio.Reader, ehlo string) (caps []string, greeting string, err error) {
	if _, err := fmt.Fprintf(conn, "EHLO %s\r\n", ehlo); err != nil {
		return nil, "", fmt.Errorf("sending EHLO: %w", err)
	}
	reply, err := readReply(br)
	if err != nil {
		return nil, "", fmt.Errorf("reading EHLO reply: %w", err)
	}
	lines := strings.Split(strings.TrimRight(reply, "\r\n"), "\n")
	for i, ln := range lines {
		ln = strings.TrimRight(ln, "\r")
		if len(ln) < 4 {
			continue
		}
		text := strings.TrimSpace(ln[4:]) // drop "250-" / "250 "
		if i == 0 {
			greeting = text
			continue
		}
		if text != "" {
			caps = append(caps, text)
		}
	}
	return caps, greeting, nil
}

// readReply reads one SMTP reply, which may span multiple lines (a hyphen after
// the code continues; a space ends).
func readReply(br *bufio.Reader) (string, error) {
	var sb strings.Builder
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			sb.WriteString(line)
		}
		if err != nil {
			if sb.Len() > 0 {
				return sb.String(), nil
			}
			return "", err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if len(trimmed) >= 4 && trimmed[3] == ' ' {
			return sb.String(), nil
		}
		if len(trimmed) < 4 { // malformed; stop to avoid hanging
			return sb.String(), nil
		}
	}
}

func tlsInfo(c *tls.Conn, host string) *TLSInfo {
	st := c.ConnectionState()
	info := &TLSInfo{
		Version: tlsVersionName(st.Version),
		Cipher:  tls.CipherSuiteName(st.CipherSuite),
	}
	if len(st.PeerCertificates) > 0 {
		leaf := st.PeerCertificates[0]
		info.Subject = leaf.Subject.CommonName
		info.Issuer = leaf.Issuer.CommonName
		info.DNSNames = leaf.DNSNames
		info.NotAfter = leaf.NotAfter
		info.DaysLeft = int(time.Until(leaf.NotAfter).Hours() / 24)
		info.HostMatch = leaf.VerifyHostname(host) == nil
	}
	return info
}

func authMechs(caps []string) []string {
	for _, c := range caps {
		if strings.HasPrefix(strings.ToUpper(c), "AUTH ") {
			return strings.Fields(c)[1:]
		}
	}
	return nil
}

func hasCap(caps []string, name string) bool {
	for _, c := range caps {
		if strings.EqualFold(strings.Fields(c)[0], name) {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimRight(s[:i], "\r")
	}
	return strings.TrimRight(s, "\r")
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	}
	return fmt.Sprintf("0x%04x", v)
}
