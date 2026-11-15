package discover

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

// buildMDNSResponse packs a realistic Bonjour AirPlay advertisement.
func buildMDNSResponse() []byte {
	m := new(dns.Msg)
	m.Response = true
	m.Answer = []dns.RR{
		&dns.PTR{
			Hdr: dns.RR_Header{Name: "_airplay._tcp.local.", Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
			Ptr: "Living Room._airplay._tcp.local.",
		},
		&dns.SRV{
			Hdr:    dns.RR_Header{Name: "Living Room._airplay._tcp.local.", Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 120},
			Target: "livingroom.local.", Port: 7000,
		},
		&dns.TXT{
			Hdr: dns.RR_Header{Name: "Living Room._airplay._tcp.local.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 120},
			Txt: []string{"model=AppleTV6,2", "deviceid=AA:BB:CC:DD:EE:FF"},
		},
		&dns.A{
			Hdr: dns.RR_Header{Name: "livingroom.local.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 120},
			A:   net.ParseIP("192.168.1.50"),
		},
	}
	b, _ := m.Pack()
	return b
}

func TestParseMDNS(t *testing.T) {
	svcs := parseMDNS([][]byte{buildMDNSResponse()})
	if len(svcs) != 1 {
		t.Fatalf("expected 1 service, got %d: %+v", len(svcs), svcs)
	}
	s := svcs[0]
	if s.Instance != "Living Room" {
		t.Errorf("instance = %q", s.Instance)
	}
	if s.Type != "_airplay._tcp" {
		t.Errorf("type = %q", s.Type)
	}
	if s.Host != "livingroom.local" || s.Port != 7000 {
		t.Errorf("host/port = %q/%d", s.Host, s.Port)
	}
	if len(s.Addrs) != 1 || s.Addrs[0] != "192.168.1.50" {
		t.Errorf("addrs = %v", s.Addrs)
	}
	if len(s.Info) != 2 {
		t.Errorf("txt = %v", s.Info)
	}
}

func TestParseMDNSSkipsMetaEnumeration(t *testing.T) {
	m := new(dns.Msg)
	m.Response = true
	m.Answer = []dns.RR{
		&dns.PTR{
			Hdr: dns.RR_Header{Name: "_services._dns-sd._udp.local.", Rrtype: dns.TypePTR, Class: dns.ClassINET},
			Ptr: "_airplay._tcp.local.",
		},
	}
	b, _ := m.Pack()
	svcs := parseMDNS([][]byte{b})
	if len(svcs) != 0 {
		t.Errorf("meta enumeration should yield no service instances, got %+v", svcs)
	}
}

func TestParseSSDP(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=1800\r\n" +
		"LOCATION: http://192.168.1.1:5000/rootDesc.xml\r\n" +
		"SERVER: Linux/3.14 UPnP/1.0 MiniUPnPd/2.1\r\n" +
		"ST: upnp:rootdevice\r\n" +
		"USN: uuid:abcd::upnp:rootdevice\r\n\r\n"
	d, ok := parseSSDP(raw, "192.168.1.1")
	if !ok {
		t.Fatal("expected a parsed device")
	}
	if d.Location != "http://192.168.1.1:5000/rootDesc.xml" {
		t.Errorf("location = %q", d.Location)
	}
	if d.Server != "Linux/3.14 UPnP/1.0 MiniUPnPd/2.1" {
		t.Errorf("server = %q", d.Server)
	}
	if d.Type != "upnp:rootdevice" {
		t.Errorf("type = %q", d.Type)
	}
}

func TestParseSSDPRejectsJunk(t *testing.T) {
	if _, ok := parseSSDP("not an ssdp packet at all", "1.2.3.4"); ok {
		t.Error("should reject non-SSDP data")
	}
	if _, ok := parseSSDP("HTTP/1.1 200 OK\r\nCACHE-CONTROL: x\r\n\r\n", "1.2.3.4"); ok {
		t.Error("should reject SSDP with no identifying headers")
	}
}

func TestDedupeDevices(t *testing.T) {
	in := []Device{
		{Address: "192.168.1.1", USN: "uuid:a::root", Type: "upnp:rootdevice"},
		{Address: "192.168.1.1", USN: "uuid:a::root", Type: "upnp:rootdevice"},
		{Address: "192.168.1.2", USN: "uuid:b::root", Type: "upnp:rootdevice"},
	}
	out := dedupeDevices(in)
	if len(out) != 2 {
		t.Errorf("expected 2 unique devices, got %d", len(out))
	}
}
