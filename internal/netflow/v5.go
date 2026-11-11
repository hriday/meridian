// Package netflow is Meridian's offline NetFlow engine: a v5 parser and a
// top-talkers aggregation. v9/IPFIX are added behind the same FlowRecord in
// slice 2. It is defensive against untrusted, attacker-reachable datagrams.
package netflow

import (
	"encoding/binary"
	"net/netip"
	"time"
)

// FlowRecord is one flow from an export packet. IPs are netip.Addr (comparable,
// used directly as a map key) and stringified only at the aggregation boundary.
type FlowRecord struct {
	Exporter         netip.Addr
	Src, Dst         netip.Addr
	SrcPort, DstPort uint16
	Proto            uint8
	Bytes, Packets   uint64
	T                time.Time
}

const (
	v5HeaderLen = 24
	v5RecordLen = 48
	v5MaxRecs   = 30
)

// ParseV5 parses a NetFlow v5 datagram into its flow records. It returns nil for
// a non-v5, short, or out-of-range-count packet (never panics). T is now.
func ParseV5(datagram []byte, exporter netip.Addr, now time.Time) []FlowRecord {
	if len(datagram) < v5HeaderLen || binary.BigEndian.Uint16(datagram[0:2]) != 5 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(datagram[2:4]))
	if count < 1 || count > v5MaxRecs || len(datagram) < v5HeaderLen+v5RecordLen*count {
		return nil
	}
	recs := make([]FlowRecord, 0, count)
	for i := 0; i < count; i++ {
		r := datagram[v5HeaderLen+v5RecordLen*i:]
		var src, dst [4]byte
		copy(src[:], r[0:4])
		copy(dst[:], r[4:8])
		recs = append(recs, FlowRecord{
			Exporter: exporter,
			Src:      netip.AddrFrom4(src),
			Dst:      netip.AddrFrom4(dst),
			Packets:  uint64(binary.BigEndian.Uint32(r[16:20])),
			Bytes:    uint64(binary.BigEndian.Uint32(r[20:24])),
			SrcPort:  binary.BigEndian.Uint16(r[32:34]),
			DstPort:  binary.BigEndian.Uint16(r[34:36]),
			Proto:    r[38],
			T:        now,
		})
	}
	return recs
}
