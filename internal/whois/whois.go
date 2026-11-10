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

// parseReferral returns the referred-to host from the first referral line
// (refer: / whois: / Registrar WHOIS Server: / ReferralServer:), case-
// insensitive, stripping a whois:// scheme and any port. "" if none.
func parseReferral(body string) string {
	prefixes := []string{"refer:", "whois:", "registrar whois server:", "referralserver:"}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		low := strings.ToLower(line)
		for _, p := range prefixes {
			if strings.HasPrefix(low, p) {
				if h := cleanHost(strings.TrimSpace(line[len(p):])); h != "" {
					return h
				}
			}
		}
	}
	return ""
}

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
// itself for an IP/ASN.
func ianaKey(object string) string {
	if net.ParseIP(object) != nil {
		return object
	}
	if i := strings.LastIndex(object, "."); i >= 0 && i < len(object)-1 {
		return object[i+1:]
	}
	return object
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
