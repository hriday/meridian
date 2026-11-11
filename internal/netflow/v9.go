package netflow

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"time"

	nf "github.com/netsampler/goflow2/v2/decoders/netflow"
)

// Parser decodes NetFlow v5/v9/IPFIX datagrams into FlowRecords. v9/IPFIX are
// template-based, so it holds a per-exporter template cache. It is only called
// from the collector's single read-loop goroutine, so the map needs no lock.
type Parser struct {
	templates map[netip.Addr]nf.NetFlowTemplateSystem
}

// maxExporters bounds the per-exporter template cache so a spoofed-source flood
// on the listening UDP port can't exhaust memory. Real deployments have a handful
// of exporters; once full, datagrams from new (likely spoofed) sources are
// dropped rather than growing the map.
var maxExporters = 4096

// NewParser returns a ready parser.
func NewParser() *Parser {
	return &Parser{templates: map[netip.Addr]nf.NetFlowTemplateSystem{}}
}

// Parse dispatches a datagram by version: v5 is parsed directly; v9/IPFIX go
// through the goflow2 decoder against this exporter's template cache.
func (p *Parser) Parse(datagram []byte, exporter netip.Addr, now time.Time) []FlowRecord {
	if len(datagram) < 2 {
		return nil
	}
	switch binary.BigEndian.Uint16(datagram[0:2]) {
	case 5:
		return ParseV5(datagram, exporter, now)
	case 9, 10:
		return p.parseTemplated(datagram, exporter, now)
	default:
		return nil
	}
}

// parseTemplated decodes a v9/IPFIX datagram. goflow2 sets the packet's FlowSets
// even when it returns a joined error (e.g. a data set whose template hasn't
// been seen yet), so we read what decoded and drop the rest. recover() guards
// against any decoder panic on adversarial input.
func (p *Parser) parseTemplated(datagram []byte, exporter netip.Addr, now time.Time) (recs []FlowRecord) {
	defer func() {
		if r := recover(); r != nil {
			recs = nil
		}
	}()
	ts := p.templates[exporter]
	if ts == nil {
		if len(p.templates) >= maxExporters {
			return nil // cache full — drop new (likely spoofed) sources
		}
		ts = nf.CreateTemplateSystem()
		p.templates[exporter] = ts
	}
	var v9 nf.NFv9Packet
	var ipfix nf.IPFIXPacket
	_ = nf.DecodeMessageVersion(bytes.NewBuffer(datagram), ts, &v9, &ipfix)

	flowsets := v9.FlowSets
	if binary.BigEndian.Uint16(datagram[0:2]) == 10 {
		flowsets = ipfix.FlowSets
	}
	for _, fs := range flowsets {
		df, ok := fs.(nf.DataFlowSet)
		if !ok {
			continue
		}
		for i := range df.Records {
			if fr, ok := flowFromFields(df.Records[i].Values, exporter, now); ok {
				recs = append(recs, fr)
			}
		}
	}
	return recs
}

// flowFromFields maps a decoded record's IANA IE fields to a FlowRecord, keeping
// only the top-talker fields. ok is false unless a source or destination address
// was present.
func flowFromFields(values []nf.DataField, exporter netip.Addr, now time.Time) (FlowRecord, bool) {
	fr := FlowRecord{Exporter: exporter, T: now}
	var hasSrc, hasDst bool
	for _, v := range values {
		if v.PenProvided { // skip vendor enterprise fields (type numbers collide with std IEs)
			continue
		}
		b, ok := v.Value.([]byte)
		if !ok {
			continue
		}
		switch v.Type {
		case 8: // sourceIPv4Address
			if a, ok := addr4(b); ok {
				fr.Src, hasSrc = a, true
			}
		case 27: // sourceIPv6Address
			if a, ok := addr16(b); ok {
				fr.Src, hasSrc = a, true
			}
		case 12: // destinationIPv4Address
			if a, ok := addr4(b); ok {
				fr.Dst, hasDst = a, true
			}
		case 28: // destinationIPv6Address
			if a, ok := addr16(b); ok {
				fr.Dst, hasDst = a, true
			}
		case 7: // sourceTransportPort
			fr.SrcPort = uint16(beUint(b))
		case 11: // destinationTransportPort
			fr.DstPort = uint16(beUint(b))
		case 4: // protocolIdentifier
			fr.Proto = uint8(beUint(b))
		case 1, 85: // octetDeltaCount / octetTotalCount
			fr.Bytes = beUint(b)
		case 2, 86: // packetDeltaCount / packetTotalCount
			fr.Packets = beUint(b)
		}
	}
	return fr, hasSrc || hasDst
}

// beUint reads a big-endian unsigned integer from up to 8 bytes.
func beUint(b []byte) uint64 {
	var v uint64
	for _, x := range b {
		v = v<<8 | uint64(x)
	}
	return v
}

func addr4(b []byte) (netip.Addr, bool) {
	if len(b) != 4 {
		return netip.Addr{}, false
	}
	return netip.AddrFrom4([4]byte{b[0], b[1], b[2], b[3]}), true
}

func addr16(b []byte) (netip.Addr, bool) {
	if len(b) != 16 {
		return netip.Addr{}, false
	}
	var a [16]byte
	copy(a[:], b)
	return netip.AddrFrom16(a), true
}
