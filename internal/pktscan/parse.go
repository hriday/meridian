// Package pktscan is a tcpdump-lite: it captures frames on an interface for a
// short window and summarizes them — top talkers, protocol mix, TCP resets and
// retransmits, DNS errors, and the TLS SNIs seen — so you can tell *what* is on
// the wire when mtr or rtp flags an anomaly. Capture needs raw-socket privileges
// (sudo / ChmodBPF) and reuses the platform-abstracted l2 package.
//
// This file holds the pure packet parsers; they take bytes and return structs,
// so the whole decode path is testable without a live NIC.
package pktscan

import (
	"encoding/binary"
	"net"
)

func ipv4Str(b []byte) string { return net.IP(b).To4().String() }
func ipv6Str(b []byte) string { return net.IP(b).String() }

// Proto is the L4 protocol of a parsed packet.
type Proto string

const (
	ProtoTCP   Proto = "tcp"
	ProtoUDP   Proto = "udp"
	ProtoICMP  Proto = "icmp"
	ProtoOther Proto = "other"
)

// TCP flag bits.
const (
	flagFIN = 1 << 0
	flagSYN = 1 << 1
	flagRST = 1 << 2
	flagACK = 1 << 4
)

// Packet is the parsed subset of one frame that the summary needs.
type Packet struct {
	Frame    int // on-wire bytes of the whole frame
	SrcIP    string
	DstIP    string
	Proto    Proto
	SrcPort  int
	DstPort  int
	TCPFlags uint8
	TCPSeq   uint32
	L4Len    int    // L4 payload length (for retransmit heuristics)
	DNSRcode int    // DNS response code, or -1 if not a DNS message
	SNI      string // TLS ClientHello server name, or ""
}

// parseFrame decodes an Ethernet frame down to L4 plus best-effort DNS/TLS. ok is
// false for frames that aren't IPv4/IPv6 or are truncated.
func parseFrame(f []byte) (Packet, bool) {
	p := Packet{Frame: len(f), DNSRcode: -1}
	if len(f) < 14 {
		return p, false
	}
	etherType := binary.BigEndian.Uint16(f[12:14])
	off := 14
	// Skip one 802.1Q VLAN tag if present.
	if etherType == 0x8100 && len(f) >= 18 {
		etherType = binary.BigEndian.Uint16(f[16:18])
		off = 18
	}

	var l4proto byte
	var l4 []byte
	switch etherType {
	case 0x0800: // IPv4
		if len(f) < off+20 {
			return p, false
		}
		ihl := int(f[off]&0x0f) * 4
		if ihl < 20 || len(f) < off+ihl {
			return p, false
		}
		l4proto = f[off+9]
		p.SrcIP = ipv4Str(f[off+12 : off+16])
		p.DstIP = ipv4Str(f[off+16 : off+20])
		l4 = f[off+ihl:]
	case 0x86dd: // IPv6
		if len(f) < off+40 {
			return p, false
		}
		l4proto = f[off+6] // next header (no extension-header walking)
		p.SrcIP = ipv6Str(f[off+8 : off+24])
		p.DstIP = ipv6Str(f[off+24 : off+40])
		l4 = f[off+40:]
	default:
		return p, false
	}

	switch l4proto {
	case 6: // TCP
		p.Proto = ProtoTCP
		if len(l4) < 20 {
			return p, true
		}
		p.SrcPort = int(binary.BigEndian.Uint16(l4[0:2]))
		p.DstPort = int(binary.BigEndian.Uint16(l4[2:4]))
		p.TCPSeq = binary.BigEndian.Uint32(l4[4:8])
		dataOff := int(l4[12]>>4) * 4
		p.TCPFlags = l4[13]
		if dataOff >= 20 && len(l4) >= dataOff {
			payload := l4[dataOff:]
			p.L4Len = len(payload)
			if sni := tlsSNI(payload); sni != "" {
				p.SNI = sni
			}
			if p.SrcPort == 53 || p.DstPort == 53 {
				if rc, ok := dnsRcode(payload); ok {
					p.DNSRcode = rc
				}
			}
		}
	case 17: // UDP
		p.Proto = ProtoUDP
		if len(l4) < 8 {
			return p, true
		}
		p.SrcPort = int(binary.BigEndian.Uint16(l4[0:2]))
		p.DstPort = int(binary.BigEndian.Uint16(l4[2:4]))
		payload := l4[8:]
		p.L4Len = len(payload)
		if p.SrcPort == 53 || p.DstPort == 53 {
			if rc, ok := dnsRcode(payload); ok {
				p.DNSRcode = rc
			}
		}
	case 1, 58: // ICMP / ICMPv6
		p.Proto = ProtoICMP
	default:
		p.Proto = ProtoOther
	}
	return p, true
}

// dnsRcode returns the response code of a DNS message (only for responses — the
// QR bit set). ok is false for queries or malformed headers.
func dnsRcode(b []byte) (int, bool) {
	if len(b) < 12 {
		return 0, false
	}
	flags := binary.BigEndian.Uint16(b[2:4])
	if flags&0x8000 == 0 { // QR=0: a query, not a response
		return 0, false
	}
	return int(flags & 0x000f), true
}

// tlsSNI extracts the server name from a TLS ClientHello, or "" if payload isn't
// one or has no SNI. It is defensive: any short/weird field yields "".
func tlsSNI(b []byte) string {
	// TLS record: type(1)=22 handshake, version(2), length(2).
	if len(b) < 5 || b[0] != 0x16 {
		return ""
	}
	h := b[5:]
	// Handshake: type(1)=1 ClientHello, length(3).
	if len(h) < 4 || h[0] != 0x01 {
		return ""
	}
	p := h[4:]
	// client_version(2) + random(32)
	if len(p) < 34 {
		return ""
	}
	p = p[34:]
	// session_id
	if len(p) < 1 || len(p) < 1+int(p[0]) {
		return ""
	}
	p = p[1+int(p[0]):]
	// cipher_suites
	if len(p) < 2 {
		return ""
	}
	cs := int(binary.BigEndian.Uint16(p[0:2]))
	if len(p) < 2+cs {
		return ""
	}
	p = p[2+cs:]
	// compression_methods
	if len(p) < 1 || len(p) < 1+int(p[0]) {
		return ""
	}
	p = p[1+int(p[0]):]
	// extensions
	if len(p) < 2 {
		return ""
	}
	extTotal := int(binary.BigEndian.Uint16(p[0:2]))
	p = p[2:]
	if len(p) < extTotal {
		extTotal = len(p)
	}
	ext := p[:extTotal]
	for len(ext) >= 4 {
		etype := binary.BigEndian.Uint16(ext[0:2])
		elen := int(binary.BigEndian.Uint16(ext[2:4]))
		if len(ext) < 4+elen {
			return ""
		}
		data := ext[4 : 4+elen]
		ext = ext[4+elen:]
		if etype != 0x0000 { // server_name
			continue
		}
		// server_name_list(2) then entry: type(1)=0 host_name, len(2), name.
		if len(data) < 5 || data[2] != 0x00 {
			return ""
		}
		nlen := int(binary.BigEndian.Uint16(data[3:5]))
		if len(data) < 5+nlen {
			return ""
		}
		return string(data[5 : 5+nlen])
	}
	return ""
}
