// Package rtp pulls RTP media streams out of captured Ethernet frames and
// computes, per stream (SSRC), the real jitter (RFC 3550), packet loss and
// sequence gaps from the actual audio — the ground truth when a call sounds
// bad, as opposed to inferring quality from a ping. Parsing and the statistics
// are pure and unit-tested; the live capture (raw L2) is wired separately.
package rtp

import (
	"encoding/binary"
	"fmt"
	"net"
	"sort"
)

// Packet is one parsed RTP packet.
type Packet struct {
	SSRC        uint32
	Seq         uint16
	Timestamp   uint32
	PayloadType uint8
	Src, Dst    string // ip:port
	RecvNanos   int64
}

// ParseFrame extracts an RTP packet from an Ethernet frame, or ok=false if the
// frame isn't plausibly RTP-over-UDP-over-IPv4.
func ParseFrame(frame []byte, recvNanos int64) (Packet, bool) {
	if len(frame) < 14 {
		return Packet{}, false
	}
	etherType := binary.BigEndian.Uint16(frame[12:14])
	off := 14
	if etherType == 0x8100 { // 802.1Q VLAN tag
		if len(frame) < 18 {
			return Packet{}, false
		}
		etherType = binary.BigEndian.Uint16(frame[16:18])
		off = 18
	}
	if etherType != 0x0800 { // IPv4
		return Packet{}, false
	}
	ip := frame[off:]
	if len(ip) < 20 || ip[0]>>4 != 4 {
		return Packet{}, false
	}
	ihl := int(ip[0]&0x0f) * 4
	if ihl < 20 || len(ip) < ihl || ip[9] != 17 { // protocol 17 = UDP
		return Packet{}, false
	}
	srcIP := net.IP(ip[12:16]).String()
	dstIP := net.IP(ip[16:20]).String()
	udp := ip[ihl:]
	if len(udp) < 8 {
		return Packet{}, false
	}
	srcPort := binary.BigEndian.Uint16(udp[0:2])
	dstPort := binary.BigEndian.Uint16(udp[2:4])
	payload := udp[8:]
	if len(payload) < 12 {
		return Packet{}, false
	}
	// RTP: version must be 2 (top two bits), and the payload type should be a
	// plausible audio/video one (static < 35 or dynamic 96-127).
	if payload[0]>>6 != 2 {
		return Packet{}, false
	}
	pt := payload[1] & 0x7f
	if pt > 34 && pt < 96 {
		return Packet{}, false
	}
	return Packet{
		SSRC:        binary.BigEndian.Uint32(payload[8:12]),
		Seq:         binary.BigEndian.Uint16(payload[2:4]),
		Timestamp:   binary.BigEndian.Uint32(payload[4:8]),
		PayloadType: pt,
		Src:         net.JoinHostPort(srcIP, fmt.Sprint(srcPort)),
		Dst:         net.JoinHostPort(dstIP, fmt.Sprint(dstPort)),
		RecvNanos:   recvNanos,
	}, true
}

// Stream is the running statistics for one SSRC.
type Stream struct {
	SSRC        uint32  `json:"ssrc"`
	Src         string  `json:"src"`
	Dst         string  `json:"dst"`
	PayloadType uint8   `json:"payloadType"`
	Codec       string  `json:"codec"`
	Packets     int     `json:"packets"`
	Lost        int     `json:"lost"`
	LossPct     float64 `json:"lossPct"`
	JitterMs    float64 `json:"jitterMs"`
	DurationMs  int     `json:"durationMs"`
}

type streamState struct {
	Stream
	firstSeq   uint16
	lastSeq    uint16
	haveSeq    bool
	firstNanos int64
	lastNanos  int64
	prevTransit float64
	jitter      float64 // in RTP timestamp units
	clock       float64
}

// Analyzer accumulates per-SSRC statistics from a sequence of packets.
type Analyzer struct {
	streams map[uint32]*streamState
}

// NewAnalyzer returns an empty analyzer.
func NewAnalyzer() *Analyzer { return &Analyzer{streams: map[uint32]*streamState{}} }

// Add folds one packet into its stream's statistics.
func (a *Analyzer) Add(p Packet) {
	s := a.streams[p.SSRC]
	if s == nil {
		s = &streamState{clock: clockRate(p.PayloadType)}
		s.SSRC = p.SSRC
		s.Src, s.Dst = p.Src, p.Dst
		s.PayloadType = p.PayloadType
		s.Codec = codecName(p.PayloadType)
		s.firstNanos = p.RecvNanos
		a.streams[p.SSRC] = s
	}
	s.Packets++
	s.lastNanos = p.RecvNanos

	// loss / reordering from the 16-bit sequence number
	if !s.haveSeq {
		s.firstSeq, s.lastSeq, s.haveSeq = p.Seq, p.Seq, true
	} else {
		gap := int(p.Seq - s.lastSeq) // wraps correctly for uint16
		if gap > 0 {
			s.Lost += gap - 1
			s.lastSeq = p.Seq
		}
	}

	// RFC 3550 interarrival jitter, in RTP timestamp units, then to ms.
	arrival := float64(p.RecvNanos) / 1e9 * s.clock
	transit := arrival - float64(p.Timestamp)
	if s.Packets > 1 {
		d := transit - s.prevTransit
		if d < 0 {
			d = -d
		}
		s.jitter += (d - s.jitter) / 16
	}
	s.prevTransit = transit
}

// Streams returns the accumulated streams, newest-busiest first.
func (a *Analyzer) Streams() []Stream {
	out := make([]Stream, 0, len(a.streams))
	for _, s := range a.streams {
		st := s.Stream
		total := st.Packets + st.Lost
		if total > 0 {
			st.LossPct = round(float64(st.Lost)/float64(total)*100, 1)
		}
		if s.clock > 0 {
			st.JitterMs = round(s.jitter/s.clock*1000, 2)
		}
		st.DurationMs = int((s.lastNanos - s.firstNanos) / 1e6)
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Packets > out[j].Packets })
	return out
}

func clockRate(pt uint8) float64 {
	switch pt {
	case 26, 31, 32, 33, 34: // JPEG / H.261 / MPV / MP2T / H.263 (video)
		return 90000
	default:
		return 8000 // audio codecs use an 8 kHz RTP clock
	}
}

func codecName(pt uint8) string {
	names := map[uint8]string{
		0: "G.711 PCMU", 3: "GSM", 4: "G.723", 8: "G.711 PCMA", 9: "G.722",
		15: "G.728", 18: "G.729", 26: "JPEG", 31: "H.261", 34: "H.263",
	}
	if n, ok := names[pt]; ok {
		return n
	}
	if pt >= 96 {
		return fmt.Sprintf("dynamic (PT %d)", pt)
	}
	return fmt.Sprintf("PT %d", pt)
}

func round(v float64, dp int) float64 {
	p := 1.0
	for i := 0; i < dp; i++ {
		p *= 10
	}
	return float64(int64(v*p+0.5)) / p
}
