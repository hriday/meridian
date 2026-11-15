package pktscan

import (
	"encoding/binary"
	"net"
	"testing"
)

// --- frame builders -------------------------------------------------------

func u16(v uint16) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, v); return b }

func ethIPv4(proto byte, src, dst string, l4 []byte) []byte {
	ip := make([]byte, 20)
	ip[0] = 0x45 // v4, IHL 5
	ip[9] = proto
	copy(ip[12:16], net.ParseIP(src).To4())
	copy(ip[16:20], net.ParseIP(dst).To4())
	binary.BigEndian.PutUint16(ip[2:4], uint16(20+len(l4)))
	eth := make([]byte, 14)
	copy(eth[12:14], u16(0x0800))
	return append(append(eth, ip...), l4...)
}

func tcpSeg(sport, dport uint16, flags byte, seq uint32, payload []byte) []byte {
	t := make([]byte, 20)
	binary.BigEndian.PutUint16(t[0:2], sport)
	binary.BigEndian.PutUint16(t[2:4], dport)
	binary.BigEndian.PutUint32(t[4:8], seq)
	t[12] = 5 << 4 // data offset 5 (20 bytes)
	t[13] = flags
	return append(t, payload...)
}

func udpSeg(sport, dport uint16, payload []byte) []byte {
	u := make([]byte, 8)
	binary.BigEndian.PutUint16(u[0:2], sport)
	binary.BigEndian.PutUint16(u[2:4], dport)
	binary.BigEndian.PutUint16(u[4:6], uint16(8+len(payload)))
	return append(u, payload...)
}

func dnsResponse(rcode byte) []byte {
	h := make([]byte, 12)
	binary.BigEndian.PutUint16(h[2:4], 0x8000|uint16(rcode)) // QR=1 + rcode
	return h
}

func clientHello(sni string) []byte {
	name := []byte(sni)
	entry := append([]byte{0x00}, append(u16(uint16(len(name))), name...)...) // type host_name + len + name
	snList := append(u16(uint16(len(entry))), entry...)
	ext := append(append(u16(0x0000), u16(uint16(len(snList)))...), snList...) // server_name ext
	exts := append(u16(uint16(len(ext))), ext...)

	body := append([]byte{0x03, 0x03}, make([]byte, 32)...) // version + random
	body = append(body, 0x00)                               // session_id len 0
	body = append(body, append(u16(2), 0x00, 0x2f)...)      // cipher_suites
	body = append(body, 0x01, 0x00)                         // compression
	body = append(body, exts...)

	hs := append([]byte{0x01, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}, body...)
	rec := append([]byte{0x16, 0x03, 0x01}, append(u16(uint16(len(hs))), hs...)...)
	return rec
}

// --- tests ----------------------------------------------------------------

func TestParseFrameTCP(t *testing.T) {
	seg := tcpSeg(44000, 443, flagSYN, 1000, nil)
	p, ok := parseFrame(ethIPv4(6, "10.0.0.5", "1.1.1.1", seg))
	if !ok || p.Proto != ProtoTCP {
		t.Fatalf("want tcp, got %+v ok=%v", p, ok)
	}
	if p.SrcIP != "10.0.0.5" || p.DstIP != "1.1.1.1" || p.SrcPort != 44000 || p.DstPort != 443 {
		t.Fatalf("addr/ports wrong: %+v", p)
	}
	if p.TCPFlags&flagSYN == 0 || p.TCPSeq != 1000 {
		t.Fatalf("tcp flags/seq wrong: %+v", p)
	}
}

func TestParseFrameDNSError(t *testing.T) {
	seg := udpSeg(53, 33000, dnsResponse(3)) // NXDOMAIN
	p, ok := parseFrame(ethIPv4(17, "1.1.1.1", "10.0.0.5", seg))
	if !ok || p.Proto != ProtoUDP {
		t.Fatalf("want udp, got %+v", p)
	}
	if p.DNSRcode != 3 {
		t.Fatalf("want DNS rcode 3, got %d", p.DNSRcode)
	}
}

func TestParseFrameTLSSNI(t *testing.T) {
	seg := tcpSeg(44001, 443, flagACK, 2000, clientHello("example.com"))
	p, ok := parseFrame(ethIPv4(6, "10.0.0.5", "93.184.216.34", seg))
	if !ok || p.SNI != "example.com" {
		t.Fatalf("want SNI example.com, got %q (ok=%v)", p.SNI, ok)
	}
}

func TestParseFrameVLAN(t *testing.T) {
	base := ethIPv4(6, "10.0.0.5", "1.1.1.1", tcpSeg(1, 2, flagRST, 0, nil))
	// Insert a VLAN tag: ethertype 0x8100, TCI, then original ethertype onward.
	vlan := append([]byte{}, base[:12]...)
	vlan = append(vlan, 0x81, 0x00, 0x00, 0x64) // 802.1Q, VLAN 100
	vlan = append(vlan, base[12:]...)           // original ethertype + payload
	p, ok := parseFrame(vlan)
	if !ok || p.Proto != ProtoTCP || p.TCPFlags&flagRST == 0 {
		t.Fatalf("VLAN frame not parsed: %+v ok=%v", p, ok)
	}
}

func TestParseFrameRejectsNonIP(t *testing.T) {
	if _, ok := parseFrame(make([]byte, 14)); ok {
		t.Fatal("empty ethertype frame should not parse")
	}
}
