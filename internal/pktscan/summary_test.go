package pktscan

import "testing"

func TestSummarize(t *testing.T) {
	s := newSummarizer()
	// 10.0.0.5 <-> 1.1.1.1: a SYN, two data segments (one a retransmit), a RST.
	s.ingest(Packet{Frame: 60, SrcIP: "10.0.0.5", DstIP: "1.1.1.1", Proto: ProtoTCP, SrcPort: 44000, DstPort: 443, TCPFlags: flagSYN})
	s.ingest(Packet{Frame: 600, SrcIP: "10.0.0.5", DstIP: "1.1.1.1", Proto: ProtoTCP, SrcPort: 44000, DstPort: 443, TCPFlags: flagACK, TCPSeq: 1, L4Len: 500})
	s.ingest(Packet{Frame: 600, SrcIP: "10.0.0.5", DstIP: "1.1.1.1", Proto: ProtoTCP, SrcPort: 44000, DstPort: 443, TCPFlags: flagACK, TCPSeq: 1, L4Len: 500}) // retransmit (seq already passed)
	s.ingest(Packet{Frame: 54, SrcIP: "1.1.1.1", DstIP: "10.0.0.5", Proto: ProtoTCP, SrcPort: 443, DstPort: 44000, TCPFlags: flagRST})
	// A DNS NXDOMAIN and a TLS SNI.
	s.ingest(Packet{Frame: 90, SrcIP: "1.1.1.1", DstIP: "10.0.0.5", Proto: ProtoUDP, SrcPort: 53, DstPort: 33000, DNSRcode: 3})
	s.ingest(Packet{Frame: 200, SrcIP: "10.0.0.5", DstIP: "93.184.216.34", Proto: ProtoTCP, SrcPort: 44001, DstPort: 443, TCPFlags: flagACK, SNI: "example.com", L4Len: 100, TCPSeq: 1})

	sum := s.finalize("en0", 5.0, 10)

	if sum.Packets != 6 {
		t.Fatalf("packets = %d, want 6", sum.Packets)
	}
	if sum.TCPSyns != 1 || sum.TCPResets != 1 || sum.Retransmits != 1 {
		t.Fatalf("syn/rst/retrans = %d/%d/%d, want 1/1/1", sum.TCPSyns, sum.TCPResets, sum.Retransmits)
	}
	if len(sum.TopTalkers) == 0 || sum.TopTalkers[0].IP != "10.0.0.5" {
		t.Fatalf("top talker wrong: %+v", sum.TopTalkers)
	}
	if len(sum.DNSErrors) != 1 || sum.DNSErrors[0].Rcode != 3 || sum.DNSErrors[0].Name != "NXDOMAIN" {
		t.Fatalf("dns errors wrong: %+v", sum.DNSErrors)
	}
	if len(sum.SNIs) != 1 || sum.SNIs[0] != "example.com" {
		t.Fatalf("snis wrong: %+v", sum.SNIs)
	}
	// Protocols sorted by bytes desc: TCP (1914) before UDP (90).
	if len(sum.Protocols) < 2 || sum.Protocols[0].Proto != ProtoTCP {
		t.Fatalf("protocols wrong: %+v", sum.Protocols)
	}
}
