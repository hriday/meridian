package arpscan

import (
	"net"
	"testing"
)

func TestBuildRequestShape(t *testing.T) {
	mac, _ := net.ParseMAC("02:00:5e:11:22:33")
	f := BuildRequest(mac, net.ParseIP("192.168.1.10"), net.ParseIP("192.168.1.20"))
	if len(f) != 42 {
		t.Fatalf("frame len = %d, want 42", len(f))
	}
	if net.HardwareAddr(f[0:6]).String() != "ff:ff:ff:ff:ff:ff" {
		t.Error("dst should be broadcast")
	}
	if net.HardwareAddr(f[6:12]).String() != mac.String() {
		t.Error("src MAC wrong")
	}
	if f[12] != 0x08 || f[13] != 0x06 {
		t.Error("ethertype should be ARP 0x0806")
	}
	if f[21] != 1 {
		t.Error("opcode should be request")
	}
	if net.IP(f[28:32]).String() != "192.168.1.10" {
		t.Errorf("sender IP = %s", net.IP(f[28:32]))
	}
	if net.IP(f[38:42]).String() != "192.168.1.20" {
		t.Errorf("target IP = %s", net.IP(f[38:42]))
	}
}

func TestParseReplyRoundTrip(t *testing.T) {
	// Craft a reply: swap a request into op=2 with a sender MAC/IP.
	reply := make([]byte, 42)
	reply[12], reply[13] = 0x08, 0x06 // ARP
	reply[20], reply[21] = 0, 2       // reply
	senderMAC, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	copy(reply[22:28], senderMAC)
	copy(reply[28:32], net.ParseIP("192.168.1.20").To4())

	ip, mac, ok := ParseReply(reply)
	if !ok {
		t.Fatal("valid ARP reply should parse")
	}
	if ip != "192.168.1.20" || mac != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("parsed ip=%s mac=%s", ip, mac)
	}
}

func TestParseReplyRejectsNonReplies(t *testing.T) {
	req := BuildRequest(net.HardwareAddr{1, 2, 3, 4, 5, 6}, net.IP{10, 0, 0, 1}, net.IP{10, 0, 0, 2})
	if _, _, ok := ParseReply(req); ok {
		t.Error("an ARP request (op=1) must not parse as a reply")
	}
	ip := make([]byte, 42)
	ip[12], ip[13] = 0x08, 0x00 // IPv4, not ARP
	if _, _, ok := ParseReply(ip); ok {
		t.Error("a non-ARP frame must not parse")
	}
	if _, _, ok := ParseReply([]byte{1, 2, 3}); ok {
		t.Error("a runt must not parse")
	}
}

func TestHostsInSlash24(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.0/24")
	hosts := hostsIn(ipnet, 100000)
	if len(hosts) != 254 {
		t.Errorf("/24 should yield 254 hosts, got %d", len(hosts))
	}
	if hosts[0].String() != "192.168.1.1" {
		t.Errorf("first host = %s, want .1 (network skipped)", hosts[0])
	}
	if hosts[len(hosts)-1].String() != "192.168.1.254" {
		t.Errorf("last host = %s, want .254 (broadcast skipped)", hosts[len(hosts)-1])
	}
}

func TestHostsInCapped(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("10.0.0.0/16")
	hosts := hostsIn(ipnet, 500)
	if len(hosts) != 500 {
		t.Errorf("a /16 capped at 500 should yield 500, got %d", len(hosts))
	}
}

func TestHostsInSlash31(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.0/31")
	hosts := hostsIn(ipnet, 100)
	if len(hosts) != 2 {
		t.Errorf("/31 should yield both addresses, got %d", len(hosts))
	}
}
