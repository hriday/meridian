package trace

import (
	"encoding/binary"
	"testing"
)

// buildQuote makes a minimal quoted payload: a 20-byte IPv4 header (protocol
// TCP) followed by 8 bytes carrying src/dst ports, as an ICMP error would quote.
func buildQuote(proto byte, srcPort, dstPort uint16, ihl int) []byte {
	b := make([]byte, ihl+8)
	b[0] = 0x40 | byte(ihl/4) // version 4, IHL
	b[9] = proto
	binary.BigEndian.PutUint16(b[ihl:], srcPort)
	binary.BigEndian.PutUint16(b[ihl+2:], dstPort)
	return b
}

func TestQuotedTCPPorts(t *testing.T) {
	q := buildQuote(6, 33456, 443, 20)
	src, dst, ok := quotedTCPPorts(q)
	if !ok || src != 33456 || dst != 443 {
		t.Fatalf("got src=%d dst=%d ok=%v, want 33456/443/true", src, dst, ok)
	}
}

func TestQuotedTCPPortsWithOptions(t *testing.T) {
	// A 24-byte IP header (IHL=6, i.e. 4 bytes of options) must still be parsed.
	q := buildQuote(6, 40000, 80, 24)
	src, dst, ok := quotedTCPPorts(q)
	if !ok || src != 40000 || dst != 80 {
		t.Fatalf("got src=%d dst=%d ok=%v, want 40000/80/true", src, dst, ok)
	}
}

func TestQuotedTCPPortsRejectsNonTCP(t *testing.T) {
	q := buildQuote(17, 33456, 443, 20) // UDP
	if _, _, ok := quotedTCPPorts(q); ok {
		t.Fatal("expected ok=false for a non-TCP quote")
	}
}

func TestQuotedTCPPortsRejectsShort(t *testing.T) {
	if _, _, ok := quotedTCPPorts([]byte{0x45, 0, 0}); ok {
		t.Fatal("expected ok=false for a too-short quote")
	}
}
