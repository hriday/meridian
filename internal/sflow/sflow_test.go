package sflow

import (
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestFrameToFlow(t *testing.T) {
	// Ethernet (type IPv4) + IPv4 (proto TCP) + TCP ports.
	frame := make([]byte, 14+20+20)
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	ip := frame[14:]
	ip[0] = 0x45 // v4, IHL 5
	ip[9] = 6    // TCP
	copy(ip[12:16], net.ParseIP("10.0.0.5").To4())
	copy(ip[16:20], net.ParseIP("1.1.1.1").To4())
	tcp := frame[34:]
	binary.BigEndian.PutUint16(tcp[0:2], 40000)
	binary.BigEndian.PutUint16(tcp[2:4], 443)

	fr, ok := frameToFlow(frame, 1500, 1000, netip.MustParseAddr("192.168.1.1"), time.Now())
	if !ok {
		t.Fatal("frame did not parse")
	}
	if fr.Src.String() != "10.0.0.5" || fr.Dst.String() != "1.1.1.1" {
		t.Fatalf("addrs: %s -> %s", fr.Src, fr.Dst)
	}
	if fr.SrcPort != 40000 || fr.DstPort != 443 || fr.Proto != 6 {
		t.Fatalf("ports/proto: %d->%d/%d", fr.SrcPort, fr.DstPort, fr.Proto)
	}
	// Sampling-rate scaling: 1500 bytes × 1000 samples.
	if fr.Bytes != 1500*1000 || fr.Packets != 1000 {
		t.Fatalf("scaling: bytes=%d packets=%d", fr.Bytes, fr.Packets)
	}
	if fr.Exporter.String() != "192.168.1.1" {
		t.Fatalf("exporter: %s", fr.Exporter)
	}
}

func TestFrameToFlowRejectsShort(t *testing.T) {
	if _, ok := frameToFlow([]byte{1, 2, 3}, 60, 1, netip.Addr{}, time.Now()); ok {
		t.Fatal("too-short frame should not parse")
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	// Must not panic; returns no records.
	if recs := Parse([]byte{0, 0, 0, 5, 0xff, 0xff}, netip.MustParseAddr("10.0.0.1"), time.Now()); recs != nil {
		t.Fatalf("garbage should yield no records, got %d", len(recs))
	}
}

// u32 appends a big-endian uint32 — sFlow is XDR (all fields 4-byte, big-endian).
func u32(b []byte, v uint32) []byte { return binary.BigEndian.AppendUint32(b, v) }

// buildSflowV5Datagram wraps one sampled Ethernet frame in a complete sFlow v5
// datagram: version → agent → one flow sample → one raw-packet-header record.
// This exercises the real goflow2 decode path end to end (not just frameToFlow).
func buildSflowV5Datagram(frame []byte, rate uint32) []byte {
	// Raw-header flow record: protocol(Ethernet=1), frameLen, stripped, origLen, header bytes.
	var rec []byte
	rec = u32(rec, 1)                 // header protocol: Ethernet
	rec = u32(rec, 1500)              // frame length (on the wire)
	rec = u32(rec, 0)                 // stripped
	rec = u32(rec, uint32(len(frame))) // original header length
	rec = append(rec, frame...)

	// Flow record wrapped in its record header (data format RAW=1, length).
	var records []byte
	records = u32(records, 1)               // flow record data format: RAW
	records = u32(records, uint32(len(rec))) // record length
	records = append(records, rec...)

	// Flow sample body.
	var sample []byte
	sample = u32(sample, 7)    // sample sequence number
	sample = u32(sample, 0)    // source id (type<<24 | value)
	sample = u32(sample, rate) // sampling rate
	sample = u32(sample, 0)    // sample pool
	sample = u32(sample, 0)    // drops
	sample = u32(sample, 0)    // input ifIndex
	sample = u32(sample, 0)    // output ifIndex
	sample = u32(sample, 1)    // flow records count
	sample = append(sample, records...)

	// Sample wrapped in its header (format FLOW=1, length).
	var samples []byte
	samples = u32(samples, 1)                  // sample format: FLOW
	samples = u32(samples, uint32(len(sample))) // sample length
	samples = append(samples, sample...)

	// Datagram header.
	var dg []byte
	dg = u32(dg, 5)                          // sFlow version
	dg = u32(dg, 1)                          // agent address type: IPv4
	dg = append(dg, net.ParseIP("192.168.1.1").To4()...) // agent IP
	dg = u32(dg, 0)                          // sub-agent id
	dg = u32(dg, 1)                          // datagram sequence number
	dg = u32(dg, 0)                          // uptime
	dg = u32(dg, 1)                          // samples count
	dg = append(dg, samples...)
	return dg
}

func TestParseDecodesRealDatagram(t *testing.T) {
	// eth(IPv4) + IPv4(TCP) + TCP ports — same shape as TestFrameToFlow.
	frame := make([]byte, 14+20+20)
	binary.BigEndian.PutUint16(frame[12:14], 0x0800)
	ip := frame[14:]
	ip[0] = 0x45
	ip[9] = 6
	copy(ip[12:16], net.ParseIP("10.0.0.5").To4())
	copy(ip[16:20], net.ParseIP("1.1.1.1").To4())
	tcp := frame[34:]
	binary.BigEndian.PutUint16(tcp[0:2], 40000)
	binary.BigEndian.PutUint16(tcp[2:4], 443)

	dg := buildSflowV5Datagram(frame, 1000)
	recs := Parse(dg, netip.MustParseAddr("203.0.113.9"), time.Now())
	if len(recs) != 1 {
		t.Fatalf("decoded %d records, want 1 (a real sFlow v5 datagram must decode)", len(recs))
	}
	r := recs[0]
	if r.Src.String() != "10.0.0.5" || r.Dst.String() != "1.1.1.1" || r.SrcPort != 40000 || r.DstPort != 443 || r.Proto != 6 {
		t.Fatalf("flow = %s:%d -> %s:%d proto %d", r.Src, r.SrcPort, r.Dst, r.DstPort, r.Proto)
	}
	if r.Bytes != 1500*1000 || r.Packets != 1000 {
		t.Fatalf("scaling: bytes=%d packets=%d", r.Bytes, r.Packets)
	}
	if r.Exporter.String() != "203.0.113.9" { // the UDP source, not the agent IP
		t.Fatalf("exporter = %s, want 203.0.113.9", r.Exporter)
	}
}
