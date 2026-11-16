// Package h3probe tests HTTP/3 (QUIC) reachability to a host: it attempts the
// real QUIC handshake on UDP/443 with ALPN "h3" AND checks whether the server
// advertises HTTP/3 via Alt-Svc on a normal HTTPS response. Together they tell
// apart "the server doesn't support h3" from "the server supports h3 but UDP is
// blocked on your path" — the common restrictive-network case.
package h3probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
)

// Result is one HTTP/3 reachability probe.
type Result struct {
	Host            string       `json:"host"`
	Port            int          `json:"port"`
	QUICReachable   bool         `json:"quicReachable"`         // the QUIC handshake succeeded over UDP
	ALPN            string       `json:"alpn"`                  // protocol negotiated in the handshake (e.g. "h3")
	QUICVersion     string       `json:"quicVersion,omitempty"` // negotiated QUIC version (v1, v2)
	TLSVersion      string       `json:"tlsVersion"`            // QUIC is always TLS 1.3
	HandshakeMillis int          `json:"handshakeMillis"`       // QUIC handshake round-trip
	AdvertisesH3    bool         `json:"advertisesH3"`          // server sent Alt-Svc advertising h3
	AltSvc          string       `json:"altSvc"`                // the Alt-Svc header, if any
	HTTPS           *HTTPSRecord `json:"https,omitempty"`       // the DNS HTTPS record (RFC 9460), if resolvable
	QUICError       string       `json:"quicError,omitempty"`
	Verdict         string       `json:"verdict"`
}

// Probe runs both checks against host:port (port 0 → 443).
func Probe(ctx context.Context, host string, port int) Result {
	if port == 0 {
		port = 443
	}
	r := Result{Host: host, Port: port}
	r.QUICReachable, r.ALPN, r.TLSVersion, r.QUICVersion, r.HandshakeMillis, r.QUICError = quicHandshake(ctx, host, port)
	r.AdvertisesH3, r.AltSvc = altSvcAdvertisesH3(ctx, host, port)
	if net.ParseIP(host) == nil { // an HTTPS record is keyed by name, not IP
		r.HTTPS = lookupHTTPS(ctx, host)
	}
	r.Verdict = verdict(r)
	return r
}

func quicHandshake(ctx context.Context, host string, port int) (ok bool, alpn, tlsVer, quicVer string, ms int, errStr string) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	hctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	tlsConf := &tls.Config{ServerName: host, NextProtos: []string{"h3"}}
	if net.ParseIP(host) != nil {
		tlsConf.InsecureSkipVerify = true // an IP target has no name to verify; we only test reachability
	}

	t0 := time.Now()
	conn, err := quic.DialAddr(hctx, addr, tlsConf, &quic.Config{HandshakeIdleTimeout: 7 * time.Second})
	if err != nil {
		return false, "", "", "", 0, err.Error()
	}
	defer conn.CloseWithError(0, "")
	cs := conn.ConnectionState()
	st := cs.TLS
	return true, st.NegotiatedProtocol, tlsVersionName(st.Version), cs.Version.String(), int(time.Since(t0).Milliseconds()), ""
}

func altSvcAdvertisesH3(ctx context.Context, host string, port int) (bool, string) {
	cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	url := "https://" + net.JoinHostPort(host, strconv.Itoa(port))
	req, err := http.NewRequestWithContext(cctx, http.MethodHead, url, nil)
	if err != nil {
		return false, ""
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()
	alt := resp.Header.Get("Alt-Svc")
	return strings.Contains(alt, "h3"), alt
}

func verdict(r Result) string {
	advertised := r.AdvertisesH3 || (r.HTTPS != nil && r.HTTPS.AdvertisesH3)
	switch {
	case r.QUICReachable && r.ALPN == "h3":
		return fmt.Sprintf("HTTP/3 works — the QUIC handshake succeeded (h3 over QUIC %s) in %d ms.", r.QUICVersion, r.HandshakeMillis)
	case r.QUICReachable:
		return fmt.Sprintf("QUIC is reachable but negotiated %q, not h3.", r.ALPN)
	case advertised:
		return "The server advertises HTTP/3 (via Alt-Svc or a DNS HTTPS record) but the QUIC handshake failed — UDP/443 is likely blocked on your path (firewall / NAT)."
	default:
		return "No HTTP/3 — the QUIC handshake failed and neither Alt-Svc nor a DNS HTTPS record advertises h3."
	}
}

func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	default:
		return fmt.Sprintf("0x%04x", v)
	}
}
