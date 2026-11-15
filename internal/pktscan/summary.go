package pktscan

import (
	"fmt"
	"sort"
)

// Talker is one host's share of the captured traffic (bytes to or from it).
type Talker struct {
	IP      string `json:"ip"`
	Packets int    `json:"packets"`
	Bytes   int    `json:"bytes"`
}

// ProtoStat is the per-protocol packet/byte count.
type ProtoStat struct {
	Proto   Proto `json:"proto"`
	Packets int   `json:"packets"`
	Bytes   int   `json:"bytes"`
}

// DNSError is a count of DNS responses carrying a given error code.
type DNSError struct {
	Rcode int    `json:"rcode"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Summary is the whole capture digest.
type Summary struct {
	Interface   string      `json:"interface"`
	Seconds     float64     `json:"seconds"`
	Packets     int         `json:"packets"`
	Bytes       int         `json:"bytes"`
	TopTalkers  []Talker    `json:"topTalkers"`
	Protocols   []ProtoStat `json:"protocols"`
	TCPSyns     int         `json:"tcpSyns"`     // connection attempts (SYN, no ACK)
	TCPResets   int         `json:"tcpResets"`   // RST seen
	Retransmits int         `json:"retransmits"` // best-effort: a TCP payload seq we'd already passed
	DNSErrors   []DNSError  `json:"dnsErrors"`
	SNIs        []string    `json:"snis"`
}

// summarizer ingests parsed packets and produces a Summary.
type summarizer struct {
	talkers map[string]*Talker
	protos  map[Proto]*ProtoStat
	flows   map[string]uint32 // flow -> next expected seq (for retransmit heuristic)
	dnsErr  map[int]int
	snis    map[string]bool

	packets, bytes            int
	syns, resets, retransmits int
}

func newSummarizer() *summarizer {
	return &summarizer{
		talkers: map[string]*Talker{},
		protos:  map[Proto]*ProtoStat{},
		flows:   map[string]uint32{},
		dnsErr:  map[int]int{},
		snis:    map[string]bool{},
	}
}

func (s *summarizer) credit(ip string, frame int) {
	if ip == "" {
		return
	}
	t := s.talkers[ip]
	if t == nil {
		t = &Talker{IP: ip}
		s.talkers[ip] = t
	}
	t.Packets++
	t.Bytes += frame
}

func (s *summarizer) ingest(p Packet) {
	s.packets++
	s.bytes += p.Frame
	s.credit(p.SrcIP, p.Frame)
	s.credit(p.DstIP, p.Frame)

	ps := s.protos[p.Proto]
	if ps == nil {
		ps = &ProtoStat{Proto: p.Proto}
		s.protos[p.Proto] = ps
	}
	ps.Packets++
	ps.Bytes += p.Frame

	if p.Proto == ProtoTCP {
		if p.TCPFlags&flagSYN != 0 && p.TCPFlags&flagACK == 0 {
			s.syns++
		}
		if p.TCPFlags&flagRST != 0 {
			s.resets++
		}
		if p.L4Len > 0 {
			key := fmt.Sprintf("%s:%d>%s:%d", p.SrcIP, p.SrcPort, p.DstIP, p.DstPort)
			if exp, seen := s.flows[key]; seen && p.TCPSeq < exp {
				s.retransmits++
			}
			if end := p.TCPSeq + uint32(p.L4Len); end > s.flows[key] {
				s.flows[key] = end
			}
		}
	}
	if p.DNSRcode > 0 {
		s.dnsErr[p.DNSRcode]++
	}
	if p.SNI != "" {
		s.snis[p.SNI] = true
	}
}

func (s *summarizer) finalize(iface string, seconds float64, topN int) Summary {
	out := Summary{
		Interface: iface, Seconds: seconds,
		Packets: s.packets, Bytes: s.bytes,
		TCPSyns: s.syns, TCPResets: s.resets, Retransmits: s.retransmits,
	}

	for _, t := range s.talkers {
		out.TopTalkers = append(out.TopTalkers, *t)
	}
	sort.Slice(out.TopTalkers, func(i, j int) bool {
		if out.TopTalkers[i].Bytes != out.TopTalkers[j].Bytes {
			return out.TopTalkers[i].Bytes > out.TopTalkers[j].Bytes
		}
		return out.TopTalkers[i].IP < out.TopTalkers[j].IP
	})
	if topN > 0 && len(out.TopTalkers) > topN {
		out.TopTalkers = out.TopTalkers[:topN]
	}

	for _, ps := range s.protos {
		out.Protocols = append(out.Protocols, *ps)
	}
	sort.Slice(out.Protocols, func(i, j int) bool {
		return out.Protocols[i].Bytes > out.Protocols[j].Bytes
	})

	for rc, n := range s.dnsErr {
		out.DNSErrors = append(out.DNSErrors, DNSError{Rcode: rc, Name: dnsRcodeName(rc), Count: n})
	}
	sort.Slice(out.DNSErrors, func(i, j int) bool {
		if out.DNSErrors[i].Count != out.DNSErrors[j].Count {
			return out.DNSErrors[i].Count > out.DNSErrors[j].Count
		}
		return out.DNSErrors[i].Rcode < out.DNSErrors[j].Rcode
	})

	for sni := range s.snis {
		out.SNIs = append(out.SNIs, sni)
	}
	sort.Strings(out.SNIs)
	return out
}

func dnsRcodeName(rc int) string {
	switch rc {
	case 1:
		return "FORMERR"
	case 2:
		return "SERVFAIL"
	case 3:
		return "NXDOMAIN"
	case 4:
		return "NOTIMP"
	case 5:
		return "REFUSED"
	default:
		return fmt.Sprintf("RCODE%d", rc)
	}
}
