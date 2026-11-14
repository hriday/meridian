// Package dhcpscan broadcasts a DHCP DISCOVER on the local segment and reports
// every server that answers — so you can confirm your DHCP server, see what it
// hands out, and catch a ROGUE server (more than one answering is a red flag).
//
// Building and parsing the BOOTP/DHCP packets is pure and fully unit-tested; the
// actual broadcast/listen (which must bind UDP port 68 and therefore needs
// elevated privileges) lives behind an injectable Transport.
package dhcpscan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
)

const (
	bootRequest = 1
	bootReply   = 2
	htypeEther  = 1
	magicCookie = 0x63825363

	optSubnetMask   = 1
	optRouter       = 3
	optDNS          = 6
	optDomainName   = 15
	optMessageType  = 53
	optServerID     = 54
	optLeaseTime    = 51
	optParamReqList = 55
	optClientID     = 61
	optEnd          = 255

	dhcpDiscover = 1
	dhcpOffer    = 2
)

// Server is one DHCP server's offer.
type Server struct {
	ServerID   string   `json:"serverID"`   // option 54, or the source IP
	Offered    string   `json:"offered"`    // yiaddr
	SubnetMask string   `json:"subnetMask"` // option 1
	Gateway    string   `json:"gateway"`    // option 3
	DNS        []string `json:"dns"`        // option 6
	DomainName string   `json:"domainName"` // option 15
	LeaseSecs  uint32   `json:"leaseSecs"`  // option 51
}

// Result is the whole scan.
type Result struct {
	Interface string   `json:"interface"`
	Servers   []Server `json:"servers"`
	Rogue     bool     `json:"rogue"` // more than one distinct server answered
}

// Transport broadcasts a DISCOVER and returns the raw OFFER packets it collects
// within its listen window. The real one binds UDP :68 (needs root); tests use a
// fake.
type Transport interface {
	Discover(discover []byte) ([][]byte, error)
}

// Scan builds a DISCOVER for mac, sends it via t, and parses the offers.
func Scan(t Transport, iface string, mac net.HardwareAddr, xid uint32) (Result, error) {
	if len(mac) != 6 {
		return Result{}, fmt.Errorf("need a 6-byte MAC address, got %d bytes", len(mac))
	}
	raw := BuildDiscover(mac, xid)
	offers, err := t.Discover(raw)
	if err != nil {
		return Result{}, err
	}

	res := Result{Interface: iface}
	seen := map[string]bool{}
	for _, pkt := range offers {
		s, ok := ParseOffer(pkt, xid)
		if !ok {
			continue
		}
		key := s.ServerID
		if key == "" {
			key = s.Offered
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		res.Servers = append(res.Servers, s)
	}
	sort.Slice(res.Servers, func(i, j int) bool { return res.Servers[i].ServerID < res.Servers[j].ServerID })
	res.Rogue = len(res.Servers) > 1
	return res, nil
}

// BuildDiscover assembles a DHCPDISCOVER BOOTP packet for mac with transaction
// id xid and the broadcast flag set.
func BuildDiscover(mac net.HardwareAddr, xid uint32) []byte {
	b := make([]byte, 240) // fixed BOOTP header + magic cookie
	b[0] = bootRequest
	b[1] = htypeEther
	b[2] = 6 // hlen
	binary.BigEndian.PutUint32(b[4:], xid)
	binary.BigEndian.PutUint16(b[10:], 0x8000) // broadcast flag
	copy(b[28:34], mac)                        // chaddr
	binary.BigEndian.PutUint32(b[236:], magicCookie)

	opts := []byte{
		optMessageType, 1, dhcpDiscover,
		optParamReqList, 5, optSubnetMask, optRouter, optDNS, optDomainName, optLeaseTime,
	}
	opts = append(opts, optClientID, 7, htypeEther)
	opts = append(opts, mac...)
	opts = append(opts, optEnd)
	return append(b, opts...)
}

// ParseOffer parses a DHCP reply and returns a Server if it's a DHCPOFFER for
// our xid.
func ParseOffer(pkt []byte, xid uint32) (Server, bool) {
	if len(pkt) < 240 {
		return Server{}, false
	}
	if pkt[0] != bootReply {
		return Server{}, false
	}
	if binary.BigEndian.Uint32(pkt[4:]) != xid {
		return Server{}, false
	}
	if binary.BigEndian.Uint32(pkt[236:]) != magicCookie {
		return Server{}, false
	}
	opts := parseOptions(pkt[240:])
	if len(opts[optMessageType]) != 1 || opts[optMessageType][0] != dhcpOffer {
		return Server{}, false
	}

	s := Server{Offered: ipString(pkt[16:20])} // yiaddr
	if v := opts[optServerID]; len(v) == 4 {
		s.ServerID = ipString(v)
	}
	if v := opts[optSubnetMask]; len(v) == 4 {
		s.SubnetMask = ipString(v)
	}
	if v := opts[optRouter]; len(v) >= 4 {
		s.Gateway = ipString(v[:4])
	}
	for v := opts[optDNS]; len(v) >= 4; v = v[4:] {
		s.DNS = append(s.DNS, ipString(v[:4]))
	}
	if v := opts[optDomainName]; len(v) > 0 {
		s.DomainName = string(v)
	}
	if v := opts[optLeaseTime]; len(v) == 4 {
		s.LeaseSecs = binary.BigEndian.Uint32(v)
	}
	return s, true
}

// parseOptions walks the TLV option area into a map of code -> value. Pad (0)
// and End (255) terminate/skip.
func parseOptions(b []byte) map[byte][]byte {
	out := map[byte][]byte{}
	for i := 0; i < len(b); {
		code := b[i]
		if code == optEnd {
			break
		}
		if code == 0 { // pad
			i++
			continue
		}
		if i+1 >= len(b) {
			break
		}
		ln := int(b[i+1])
		if i+2+ln > len(b) {
			break
		}
		out[code] = b[i+2 : i+2+ln]
		i += 2 + ln
	}
	return out
}

func ipString(b []byte) string {
	if len(b) != 4 {
		return ""
	}
	return net.IPv4(b[0], b[1], b[2], b[3]).String()
}

// ErrNeedsPrivilege is returned by the real transport when binding UDP :68 is
// denied — the caller turns it into a "run elevated" message.
var ErrNeedsPrivilege = errors.New("DHCP discovery needs elevated privileges to bind UDP port 68 — run with sudo, or launch Meridian elevated")
