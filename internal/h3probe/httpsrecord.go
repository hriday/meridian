package h3probe

import (
	"context"
	"strings"

	mdns "github.com/miekg/dns"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/sysconf"
)

// HTTPSRecord is a domain's DNS HTTPS record (RFC 9460). It advertises how to
// reach the service before any connection — which ALPNs it offers (so whether it
// does HTTP/3), an alternate port, address hints, and an Encrypted ClientHello
// config. It's the DNS-side counterpart to the Alt-Svc header.
type HTTPSRecord struct {
	Found        bool     `json:"found"`
	Priority     uint16   `json:"priority"`
	Target       string   `json:"target,omitempty"` // "." means "this same name"
	ALPN         []string `json:"alpn,omitempty"`
	Port         int      `json:"port,omitempty"`
	IPv4Hint     []string `json:"ipv4Hint,omitempty"`
	IPv6Hint     []string `json:"ipv6Hint,omitempty"`
	ECH          bool     `json:"ech"`          // an Encrypted ClientHello config is published
	AdvertisesH3 bool     `json:"advertisesH3"` // the ALPN set includes "h3"
}

// httpsQueryFn is the DNS exchange, behind a var so tests can inject records.
var httpsQueryFn = dnsx.QueryMsg

// lookupHTTPS queries host's HTTPS record and parses its SvcParams. It returns
// nil only when the query itself fails; a successful query with no HTTPS record
// yields &HTTPSRecord{Found: false}.
func lookupHTTPS(ctx context.Context, host string) *HTTPSRecord {
	server := "1.1.1.1"
	if f := sysconf.Resolvers(); len(f.Values) > 0 {
		server = f.Values[0]
	}
	msg, err := httpsQueryFn(ctx, host, "HTTPS", server)
	if err != nil || msg == nil {
		return nil
	}
	for _, rr := range msg.Answer {
		switch v := rr.(type) {
		case *mdns.HTTPS:
			return parseSVCB(v.Priority, v.Target, v.Value)
		case *mdns.SVCB:
			return parseSVCB(v.Priority, v.Target, v.Value)
		}
	}
	return &HTTPSRecord{Found: false}
}

func parseSVCB(priority uint16, target string, params []mdns.SVCBKeyValue) *HTTPSRecord {
	r := &HTTPSRecord{Found: true, Priority: priority, Target: strings.TrimSuffix(target, ".")}
	for _, kv := range params {
		switch v := kv.(type) {
		case *mdns.SVCBAlpn:
			r.ALPN = v.Alpn
		case *mdns.SVCBPort:
			r.Port = int(v.Port)
		case *mdns.SVCBIPv4Hint:
			for _, ip := range v.Hint {
				r.IPv4Hint = append(r.IPv4Hint, ip.String())
			}
		case *mdns.SVCBIPv6Hint:
			for _, ip := range v.Hint {
				r.IPv6Hint = append(r.IPv6Hint, ip.String())
			}
		case *mdns.SVCBECHConfig:
			r.ECH = len(v.ECH) > 0
		}
	}
	for _, a := range r.ALPN {
		if a == "h3" {
			r.AdvertisesH3 = true
		}
	}
	return r
}
