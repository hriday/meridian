package rtp

import (
	"encoding/binary"
	"net"
	"testing"
)

func rtpFrame(ssrc uint32, seq uint16, ts uint32, pt uint8) []byte {
	f := make([]byte, 14+20+8+12)
	binary.BigEndian.PutUint16(f[12:], 0x0800) // IPv4
	ip := f[14:]
	ip[0] = 0x45 // v4, IHL 5
	ip[9] = 17   // UDP
	copy(ip[12:16], net.ParseIP("10.0.0.1").To4())
	copy(ip[16:20], net.ParseIP("10.0.0.2").To4())
	udp := f[34:]
	binary.BigEndian.PutUint16(udp[0:], 10000)
	binary.BigEndian.PutUint16(udp[2:], 10002)
	r := f[42:]
	r[0] = 0x80 // RTP v2
	r[1] = pt
	binary.BigEndian.PutUint16(r[2:], seq)
	binary.BigEndian.PutUint32(r[4:], ts)
	binary.BigEndian.PutUint32(r[8:], ssrc)
	return f
}

func TestParseFrame(t *testing.T) {
	p, ok := ParseFrame(rtpFrame(0xABCD, 100, 1600, 0), 0)
	if !ok {
		t.Fatal("a valid RTP frame should parse")
	}
	if p.SSRC != 0xABCD || p.Seq != 100 || p.Timestamp != 1600 || p.PayloadType != 0 {
		t.Errorf("parsed wrong: %+v", p)
	}
	if p.Src != "10.0.0.1:10000" || p.Dst != "10.0.0.2:10002" {
		t.Errorf("addrs wrong: %s -> %s", p.Src, p.Dst)
	}
}

func TestParseRejectsNonRTP(t *testing.T) {
	tcp := rtpFrame(1, 1, 1, 0)
	tcp[14+9] = 6 // protocol TCP
	if _, ok := ParseFrame(tcp, 0); ok {
		t.Error("a TCP frame should not parse as RTP")
	}
	badver := rtpFrame(1, 1, 1, 0)
	badver[42] = 0x40 // RTP version 1
	if _, ok := ParseFrame(badver, 0); ok {
		t.Error("RTP version != 2 should not parse")
	}
}

func TestAnalyzerLossAndCodec(t *testing.T) {
	a := NewAnalyzer()
	for _, seq := range []uint16{1, 2, 3, 5, 6} { // 4 is missing
		p, _ := ParseFrame(rtpFrame(0x1, seq, uint32(seq)*160, 0), int64(seq)*20_000_000)
		a.Add(p)
	}
	ss := a.Streams()
	if len(ss) != 1 {
		t.Fatalf("want 1 stream, got %d", len(ss))
	}
	if ss[0].Lost != 1 {
		t.Errorf("lost = %d, want 1", ss[0].Lost)
	}
	if ss[0].Codec != "G.711 PCMU" {
		t.Errorf("codec = %q, want G.711 PCMU", ss[0].Codec)
	}
	if ss[0].Packets != 5 {
		t.Errorf("packets = %d, want 5", ss[0].Packets)
	}
}

func TestAnalyzerJitter(t *testing.T) {
	// Evenly paced: timestamps and arrivals both advance 160 units / 20 ms → ~0 jitter.
	even := NewAnalyzer()
	for i := 0; i < 10; i++ {
		p, _ := ParseFrame(rtpFrame(0x2, uint16(i), uint32(i)*160, 8), int64(i)*20_000_000)
		even.Add(p)
	}
	if j := even.Streams()[0].JitterMs; j > 0.5 {
		t.Errorf("even arrival should have ~0 jitter, got %.2f", j)
	}

	// Uneven arrival (sender paced evenly) → non-zero jitter.
	uneven := NewAnalyzer()
	arr := []int64{0, 20, 55, 60, 95, 100} // ms, jittery
	for i, ms := range arr {
		p, _ := ParseFrame(rtpFrame(0x3, uint16(i), uint32(i)*160, 8), ms*1_000_000)
		uneven.Add(p)
	}
	if j := uneven.Streams()[0].JitterMs; j <= 0 {
		t.Errorf("uneven arrival should have jitter > 0, got %.2f", j)
	}
}
