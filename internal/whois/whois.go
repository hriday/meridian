// Package whois queries port-43 Whois for a domain, IP, or ASN. It discovers the
// authoritative server the way the whois command does — ask whois.iana.org,
// follow its referral to the registry/RIR, and optionally follow one more to the
// registrar — then returns the raw record and the chain of servers consulted.
// The query carries only the object, no user identity. parseReferral is pure;
// the dial is behind a seam for offline tests. The CLI and Wails controller are
// thin adapters over Query.
package whois

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"time"
)

// Result is the outcome of a whois lookup.
type Result struct {
	Object string   `json:"object"`
	Server string   `json:"server"` // the server that produced Raw
	Chain  []string `json:"chain"`  // every server consulted, in order
	Raw    string   `json:"raw"`
}

// dialWhois opens a TCP connection to a whois server (addr is host:43). A
// package var so tests can point it at local listeners.
var dialWhois = func(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{}
	return d.DialContext(ctx, "tcp", addr)
}

// maxHops is how many referrals to follow after the IANA step
// (registry, then registrar).
const maxHops = 2

// Query looks up object, discovering the authoritative server via IANA and
// following referrals up to maxHops. It returns the most authoritative record
// reached and the server chain.
func Query(ctx context.Context, object string, timeout time.Duration) (Result, error) {
	object = strings.TrimSpace(object)
	if object == "" {
		return Result{}, errors.New("no object to look up")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res := Result{Object: object}

	server := "whois.iana.org"
	body, err := queryServer(ctx, server, ianaKey(object))
	if err != nil {
		return Result{}, err
	}
	res.Chain = append(res.Chain, server)
	res.Server, res.Raw = server, body

	for hop := 0; hop < maxHops; hop++ {
		next := parseReferral(res.Raw)
		if next == "" || next == res.Server || contains(res.Chain, next) {
			break
		}
		b, err := queryServer(ctx, next, object)
		if err != nil {
			break // referral target unreachable — keep the best record so far
		}
		res.Chain = append(res.Chain, next)
		res.Server, res.Raw = next, b
	}
	return res, nil
}

// queryServer performs one port-43 exchange: send object + CRLF, read the whole
// response (the server closes when done).
func queryServer(ctx context.Context, server, object string) (string, error) {
	conn, err := dialWhois(ctx, net.JoinHostPort(server, "43"))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	if _, err := conn.Write([]byte(object + "\r\n")); err != nil {
		return "", err
	}
	// Many whois servers abort the connection (RST) right after sending the
	// record, which surfaces as a read error even though the full response
	// arrived. Keep whatever we read; only fail when nothing came back.
	b, err := io.ReadAll(conn)
	if len(b) == 0 && err != nil {
		return "", err
	}
	return string(b), nil
}

// parseReferral returns the referred-to host from a referral line, "" if none.
// Prefixes are tried in priority order so a registry record's real registrar
// referral wins over a stray "whois:" comment line elsewhere in the body; the
// root-referral prefixes (refer:/whois:) are the fallback, which is what IANA
// responses carry. Case-insensitive; a whois:// scheme and any port are stripped.
func parseReferral(body string) string {
	lines := strings.Split(body, "\n")
	for _, p := range []string{"registrar whois server:", "referralserver:", "refer:", "whois:"} {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), p) {
				if h := cleanHost(strings.TrimSpace(line[len(p):])); h != "" {
					return h
				}
			}
		}
	}
	return ""
}

// cleanHost extracts the hostname from a referral value, dropping a whois://
// scheme, trailing text, and any port. (The port is dropped because every hop is
// queried on 43; a non-43 referral — e.g. an rwhois ReferralServer — is not
// followed, which is out of scope for registry/RIR → registrar.)
func cleanHost(v string) string {
	v = strings.TrimPrefix(v, "whois://")
	if i := strings.IndexAny(v, " \t"); i >= 0 {
		v = v[:i]
	}
	if h, _, err := net.SplitHostPort(v); err == nil {
		v = h
	}
	return v
}

// ianaKey is what to ask IANA: a domain's TLD (its root-db key), or the object
// itself for an IP/ASN. A trailing FQDN dot is stripped, and a last label is only
// treated as a TLD when it is alphabetic (so a CIDR like 1.0.0.0/24 isn't keyed
// to a bogus "0/24").
func ianaKey(object string) string {
	object = strings.TrimRight(object, ".")
	if net.ParseIP(object) != nil {
		return object
	}
	if i := strings.LastIndex(object, "."); i >= 0 && i < len(object)-1 {
		if tld := object[i+1:]; isAlpha(tld) {
			return tld
		}
	}
	return object
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return true
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
