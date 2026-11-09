// Package asnmap annotates an IP address with its origin AS number and the
// owner's name, using Team Cymru's IP-to-ASN DNS service (reusing the project's
// DNS engine). Non-global addresses are skipped without any query.
package asnmap

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/hriday/meridian/internal/dnsx"
)

// ASInfo is the network an IP belongs to.
type ASInfo struct {
	ASN   string `json:"asn"`
	Owner string `json:"owner"`
}

// QueryFunc is the DNS query surface Lookup needs; it matches dnsx.Query so the
// caller passes that and tests pass a fake.
type QueryFunc func(ctx context.Context, name, rtype, server string) (dnsx.Result, error)

// CymruResolver is the recursive resolver used for the Cymru lookups.
var CymruResolver = "1.1.1.1"

// Lookup returns the AS number and owner for ip. A non-global ip (private,
// loopback, link-local, multicast) returns an empty ASInfo without querying.
// If the AS-name lookup fails, the ASN is still returned with an empty Owner.
func Lookup(ctx context.Context, q QueryFunc, ip string) (ASInfo, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || !isGlobal(addr) {
		return ASInfo{}, nil
	}

	res, err := q(ctx, cymruOriginName(addr), "TXT", CymruResolver)
	if err != nil {
		return ASInfo{}, err
	}
	asn, _ := parseOriginTXT(firstTXT(res))
	if asn == "" {
		return ASInfo{}, nil
	}
	info := ASInfo{ASN: asn}

	if nameRes, err := q(ctx, "AS"+asn+".asn.cymru.com", "TXT", CymruResolver); err == nil {
		info.Owner = parseASNameTXT(firstTXT(nameRes))
	}
	return info, nil
}

func isGlobal(a netip.Addr) bool {
	return a.IsValid() && !a.IsPrivate() && !a.IsLoopback() &&
		!a.IsLinkLocalUnicast() && !a.IsLinkLocalMulticast() &&
		!a.IsMulticast() && !a.IsUnspecified()
}

func firstTXT(r dnsx.Result) string {
	for _, a := range r.Answers {
		if a.Type == "TXT" {
			return a.Value
		}
	}
	return ""
}

func cymruOriginName(a netip.Addr) string {
	if a.Is4() {
		b := a.As4()
		return fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com", b[3], b[2], b[1], b[0])
	}
	b := a.As16()
	var sb strings.Builder
	for i := 15; i >= 0; i-- {
		sb.WriteString(fmt.Sprintf("%x.%x.", b[i]&0x0f, b[i]>>4))
	}
	sb.WriteString("origin6.asn.cymru.com")
	return sb.String()
}

// parseOriginTXT reads Cymru's origin record: "13335 | 104.16.0.0/13 | US | …".
func parseOriginTXT(txt string) (asn, prefix string) {
	parts := strings.Split(unquote(txt), "|")
	if len(parts) < 2 {
		return "", ""
	}
	if f := strings.Fields(strings.TrimSpace(parts[0])); len(f) > 0 {
		asn = f[0] // first ASN when several origin the prefix
	}
	return asn, strings.TrimSpace(parts[1])
}

// parseASNameTXT reads Cymru's AS record: "13335 | US | arin | … | NAME, CC".
func parseASNameTXT(txt string) string {
	parts := strings.Split(unquote(txt), "|")
	return strings.TrimSpace(parts[len(parts)-1])
}

func unquote(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"`)
}
