package dhcpscan

import (
	"encoding/binary"
	"net"
	"testing"
)

var testMAC = net.HardwareAddr{0x02, 0x00, 0x5e, 0x11, 0x22, 0x33}

// buildOffer crafts a DHCPOFFER for xid from serverID, offering yiaddr.
func buildOffer(xid uint32, serverID, yiaddr, gw string, dns []string, lease uint32) []byte {
	b := make([]byte, 240)
	b[0] = bootReply
	b[1] = htypeEther
	b[2] = 6
	binary.BigEndian.PutUint32(b[4:], xid)
	copy(b[16:20], net.ParseIP(yiaddr).To4()) // yiaddr
	binary.BigEndian.PutUint32(b[236:], magicCookie)

	opts := []byte{optMessageType, 1, dhcpOffer}
	opts = append(opts, optServerID, 4)
	opts = append(opts, net.ParseIP(serverID).To4()...)
	opts = append(opts, optRouter, 4)
	opts = append(opts, net.ParseIP(gw).To4()...)
	opts = append(opts, optDNS, byte(4*len(dns)))
	for _, d := range dns {
		opts = append(opts, net.ParseIP(d).To4()...)
	}
	lb := make([]byte, 4)
	binary.BigEndian.PutUint32(lb, lease)
	opts = append(opts, optLeaseTime, 4)
	opts = append(opts, lb...)
	opts = append(opts, optEnd)
	return append(b, opts...)
}

type fakeTransport struct {
	offers   [][]byte
	gotDisco []byte
}

func (f *fakeTransport) Discover(d []byte) ([][]byte, error) {
	f.gotDisco = d
	return f.offers, nil
}

func TestBuildDiscoverShape(t *testing.T) {
	pkt := BuildDiscover(testMAC, 0xdeadbeef)
	if pkt[0] != bootRequest {
		t.Errorf("op = %d, want bootRequest", pkt[0])
	}
	if binary.BigEndian.Uint32(pkt[4:]) != 0xdeadbeef {
		t.Error("xid not set")
	}
	if binary.BigEndian.Uint16(pkt[10:]) != 0x8000 {
		t.Error("broadcast flag not set")
	}
	if net.HardwareAddr(pkt[28:34]).String() != testMAC.String() {
		t.Errorf("chaddr = %s, want %s", net.HardwareAddr(pkt[28:34]), testMAC)
	}
	if binary.BigEndian.Uint32(pkt[236:]) != magicCookie {
		t.Error("magic cookie missing")
	}
	opts := parseOptions(pkt[240:])
	if len(opts[optMessageType]) != 1 || opts[optMessageType][0] != dhcpDiscover {
		t.Error("message type is not DISCOVER")
	}
}

func TestScanParsesSingleServer(t *testing.T) {
	xid := uint32(0x11223344)
	ft := &fakeTransport{offers: [][]byte{
		buildOffer(xid, "192.168.1.1", "192.168.1.50", "192.168.1.1", []string{"192.168.1.1", "8.8.8.8"}, 86400),
	}}
	res, err := Scan(ft, "en0", testMAC, xid)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(res.Servers) != 1 {
		t.Fatalf("want 1 server, got %d", len(res.Servers))
	}
	s := res.Servers[0]
	if s.ServerID != "192.168.1.1" || s.Offered != "192.168.1.50" || s.Gateway != "192.168.1.1" {
		t.Errorf("parsed wrong: %+v", s)
	}
	if len(s.DNS) != 2 || s.DNS[1] != "8.8.8.8" {
		t.Errorf("DNS parsed wrong: %v", s.DNS)
	}
	if s.LeaseSecs != 86400 {
		t.Errorf("lease = %d, want 86400", s.LeaseSecs)
	}
	if res.Rogue {
		t.Error("one server should not be flagged rogue")
	}
}

func TestScanFlagsRogue(t *testing.T) {
	xid := uint32(7)
	ft := &fakeTransport{offers: [][]byte{
		buildOffer(xid, "192.168.1.1", "192.168.1.50", "192.168.1.1", []string{"192.168.1.1"}, 3600),
		buildOffer(xid, "192.168.1.66", "10.0.0.5", "10.0.0.1", []string{"10.0.0.1"}, 600), // rogue!
	}}
	res, _ := Scan(ft, "en0", testMAC, xid)
	if len(res.Servers) != 2 {
		t.Fatalf("want 2 servers, got %d", len(res.Servers))
	}
	if !res.Rogue {
		t.Error("two distinct DHCP servers must be flagged rogue")
	}
}

func TestScanDedupsAndIgnoresWrongXID(t *testing.T) {
	xid := uint32(100)
	ft := &fakeTransport{offers: [][]byte{
		buildOffer(xid, "192.168.1.1", "192.168.1.50", "192.168.1.1", nil, 3600),
		buildOffer(xid, "192.168.1.1", "192.168.1.51", "192.168.1.1", nil, 3600), // dup server
		buildOffer(999, "10.0.0.1", "10.0.0.9", "10.0.0.1", nil, 3600),           // wrong xid, ignored
	}}
	res, _ := Scan(ft, "en0", testMAC, xid)
	if len(res.Servers) != 1 {
		t.Errorf("want 1 deduped server, got %d: %+v", len(res.Servers), res.Servers)
	}
	if res.Rogue {
		t.Error("same server twice is not rogue")
	}
}
