package ndp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
)

var errUnsupported = errors.New("IPv6 neighbor discovery isn't supported on this OS")

// RA flag bits (RFC 4861 §4.2) and option types.
const (
	raFlagManaged = 0x80
	raFlagOther   = 0x40

	optSourceLLAddr = 1
	optPrefixInfo   = 3
	optMTU          = 5
	optRDNSS        = 25
	optDNSSL        = 31

	prefixFlagOnLink = 0x80
	prefixFlagAuto   = 0x40
)

// parseRA parses a Router Advertisement message body — the bytes after the
// 4-byte ICMPv6 header, i.e. starting at Cur Hop Limit (RFC 4861 §4.2). source
// and iface label where it came from.
func parseRA(body []byte, source, iface string) (RouterAdvert, error) {
	if len(body) < 12 {
		return RouterAdvert{}, fmt.Errorf("short RA: %d bytes", len(body))
	}
	ra := RouterAdvert{
		Router:         source,
		Interface:      iface,
		HopLimit:       body[0],
		Managed:        body[1]&raFlagManaged != 0,
		OtherConfig:    body[1]&raFlagOther != 0,
		RouterLifetime: binary.BigEndian.Uint16(body[2:4]),
		ReachableTime:  binary.BigEndian.Uint32(body[4:8]),
		RetransTimer:   binary.BigEndian.Uint32(body[8:12]),
	}
	// Options: TLV where length is in units of 8 bytes and includes the header.
	for off := 12; off+2 <= len(body); {
		otype := body[off]
		olen := int(body[off+1]) * 8
		if olen == 0 || off+olen > len(body) {
			break // malformed length; stop rather than loop/overrun
		}
		opt := body[off : off+olen]
		switch otype {
		case optSourceLLAddr:
			if len(opt) >= 8 {
				ra.SourceMAC = net.HardwareAddr(opt[2:8]).String()
			}
		case optPrefixInfo:
			if p, ok := parsePrefixOption(opt); ok {
				ra.Prefixes = append(ra.Prefixes, p)
			}
		case optMTU:
			if len(opt) >= 8 {
				ra.MTU = binary.BigEndian.Uint32(opt[4:8])
			}
		case optRDNSS:
			ra.DNS = append(ra.DNS, parseRDNSS(opt)...)
		case optDNSSL:
			ra.DNSSearch = append(ra.DNSSearch, parseDNSSL(opt)...)
		}
		off += olen
	}
	return ra, nil
}

func parsePrefixOption(opt []byte) (Prefix, bool) {
	if len(opt) < 32 {
		return Prefix{}, false
	}
	plen := int(opt[2])
	flags := opt[3]
	addr := net.IP(append([]byte(nil), opt[16:32]...))
	return Prefix{
		Prefix:            fmt.Sprintf("%s/%d", addr.String(), plen),
		OnLink:            flags&prefixFlagOnLink != 0,
		Autonomous:        flags&prefixFlagAuto != 0,
		ValidLifetime:     binary.BigEndian.Uint32(opt[4:8]),
		PreferredLifetime: binary.BigEndian.Uint32(opt[8:12]),
	}, true
}

// parseRDNSS extracts the recursive DNS server addresses from an RDNSS option
// (RFC 8106): 2 reserved + 4 lifetime bytes, then N 16-byte addresses.
func parseRDNSS(opt []byte) []string {
	if len(opt) < 16 {
		return nil
	}
	var out []string
	for p := 8; p+16 <= len(opt); p += 16 {
		out = append(out, net.IP(append([]byte(nil), opt[p:p+16]...)).String())
	}
	return out
}

// parseDNSSL extracts the DNS search domains from a DNSSL option (RFC 8106):
// reserved+lifetime, then RFC 1035 label-encoded domains, zero-padded.
func parseDNSSL(opt []byte) []string {
	if len(opt) < 8 {
		return nil
	}
	var out []string
	var labels []string
	for p := 8; p < len(opt); {
		l := int(opt[p])
		p++
		if l == 0 {
			if len(labels) > 0 {
				out = append(out, strings.Join(labels, "."))
				labels = nil
			}
			continue
		}
		if p+l > len(opt) {
			break
		}
		labels = append(labels, string(opt[p:p+l]))
		p += l
	}
	return out
}
