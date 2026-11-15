// Package sflow decodes sFlow v5 datagrams (RFC 3176 / sflow.org) into the same
// FlowRecord model the NetFlow collector uses, so sampled flows feed the shared
// top-talkers view. sFlow carries *sampled* packet headers (1-in-N), so a
// record's bytes/packets are scaled by the sample's sampling rate to estimate
// the real traffic. Decoding uses goflow2; the Ethernet/IP/L4 header parse is
// local and pure (so it's testable without a live exporter).
package sflow

import (
	"bytes"
	"encoding/binary"
	"net/netip"
	"time"

	"github.com/hriday/meridian/internal/netflow"
	gsflow "github.com/netsampler/goflow2/v2/decoders/sflow"
)

// sampledHeaderEthernet is the sFlow header-protocol value for ISO 8023 Ethernet.
const sampledHeaderEthernet = 1

// Parse decodes an sFlow datagram from src into FlowRecords (one per sampled
// Ethernet header). A decoder panic on adversarial input is contained.
func Parse(datagram []byte, src netip.Addr, now time.Time) (out []netflow.FlowRecord) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()

	// DecodeMessageVersion consumes the leading sFlow version word (5) and then
	// the datagram body; DecodeMessage alone expects the version already stripped
	// and would misread it as the agent address type.
	var pkt gsflow.Packet
	if err := gsflow.DecodeMessageVersion(bytes.NewBuffer(datagram), &pkt); err != nil {
		return nil
	}
	for _, s := range pkt.Samples {
		var rate uint32
		var records []gsflow.FlowRecord
		switch fs := s.(type) {
		case gsflow.FlowSample:
			rate, records = fs.SamplingRate, fs.Records
		case gsflow.ExpandedFlowSample:
			rate, records = fs.SamplingRate, fs.Records
		default:
			continue // counter samples etc.
		}
		if rate == 0 {
			rate = 1
		}
		for _, rec := range records {
			var sh gsflow.SampledHeader
			switch h := rec.Data.(type) {
			case gsflow.SampledHeader:
				sh = h
			case *gsflow.SampledHeader:
				sh = *h
			default:
				continue
			}
			if sh.Protocol != sampledHeaderEthernet {
				continue
			}
			if fr, ok := frameToFlow(sh.HeaderData, int(sh.FrameLength), rate, src, now); ok {
				out = append(out, fr)
			}
		}
	}
	return out
}

// frameToFlow parses an Ethernet frame header into a FlowRecord, scaling bytes
// and packets by the sampling rate.
func frameToFlow(frame []byte, frameLen int, rate uint32, exporter netip.Addr, now time.Time) (netflow.FlowRecord, bool) {
	if len(frame) < 14 {
		return netflow.FlowRecord{}, false
	}
	etype := binary.BigEndian.Uint16(frame[12:14])
	off := 14
	if etype == 0x8100 && len(frame) >= 18 { // 802.1Q VLAN
		etype = binary.BigEndian.Uint16(frame[16:18])
		off = 18
	}

	var src, dst netip.Addr
	var l4proto byte
	var l4 []byte
	switch etype {
	case 0x0800: // IPv4
		if len(frame) < off+20 {
			return netflow.FlowRecord{}, false
		}
		ihl := int(frame[off]&0x0f) * 4
		if ihl < 20 || len(frame) < off+ihl {
			return netflow.FlowRecord{}, false
		}
		l4proto = frame[off+9]
		src, _ = netip.AddrFromSlice(frame[off+12 : off+16])
		dst, _ = netip.AddrFromSlice(frame[off+16 : off+20])
		l4 = frame[off+ihl:]
	case 0x86dd: // IPv6
		if len(frame) < off+40 {
			return netflow.FlowRecord{}, false
		}
		l4proto = frame[off+6]
		src, _ = netip.AddrFromSlice(frame[off+8 : off+24])
		dst, _ = netip.AddrFromSlice(frame[off+24 : off+40])
		l4 = frame[off+40:]
	default:
		return netflow.FlowRecord{}, false
	}

	fr := netflow.FlowRecord{
		Exporter: exporter,
		Src:      src.Unmap(),
		Dst:      dst.Unmap(),
		Proto:    l4proto,
		Bytes:    uint64(frameLen) * uint64(rate),
		Packets:  uint64(rate),
		T:        now,
	}
	if (l4proto == 6 || l4proto == 17) && len(l4) >= 4 { // TCP / UDP
		fr.SrcPort = binary.BigEndian.Uint16(l4[0:2])
		fr.DstPort = binary.BigEndian.Uint16(l4[2:4])
	}
	return fr, true
}
