package netflow

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"

	nf "github.com/netsampler/goflow2/v2/decoders/netflow"
	"github.com/stretchr/testify/require"
)

func be16(v uint16) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, v); return b }
func be32(v uint32) []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, v); return b }

// standard 5-tuple + octet/packet-count template (id 256).
var v9Fields = [][2]uint16{{8, 4}, {12, 4}, {7, 2}, {11, 2}, {4, 1}, {1, 4}, {2, 4}}

func v9Header(count uint16) []byte {
	h := make([]byte, 20)
	binary.BigEndian.PutUint16(h[0:2], 9)
	binary.BigEndian.PutUint16(h[2:4], count)
	return h
}

func v9TemplateFS() []byte {
	body := append(be16(256), be16(uint16(len(v9Fields)))...)
	for _, f := range v9Fields {
		body = append(body, be16(f[0])...)
		body = append(body, be16(f[1])...)
	}
	out := append(be16(0), be16(uint16(4+len(body)))...) // flowset id 0 (template) + length
	return append(out, body...)
}

func v9DataFS(src, dst string, sport, dport uint16, proto uint8, bytes, pkts uint32) []byte {
	s := netip.MustParseAddr(src).As4()
	d := netip.MustParseAddr(dst).As4()
	rec := append(append(s[:], d[:]...), be16(sport)...)
	rec = append(rec, be16(dport)...)
	rec = append(rec, proto)
	rec = append(rec, be32(bytes)...)
	rec = append(rec, be32(pkts)...)
	out := append(be16(256), be16(uint16(4+len(rec)))...) // flowset id 256 (data) + length
	return append(out, rec...)
}

func TestFlowFromFields(t *testing.T) {
	exp := netip.MustParseAddr("10.0.0.254")
	fields := []nf.DataField{
		{Type: 8, Value: []byte{10, 0, 0, 9}},
		{Type: 12, Value: []byte{10, 0, 0, 1}},
		{Type: 7, Value: be16(50000)},
		{Type: 11, Value: be16(443)},
		{Type: 4, Value: []byte{6}},
		{Type: 1, Value: be32(120000)},
		{Type: 2, Value: be32(100)},
	}
	fr, ok := flowFromFields(fields, exp, time.Now())
	require.True(t, ok)
	require.Equal(t, "10.0.0.9", fr.Src.String())
	require.Equal(t, "10.0.0.1", fr.Dst.String())
	require.EqualValues(t, 50000, fr.SrcPort)
	require.EqualValues(t, 443, fr.DstPort)
	require.EqualValues(t, 6, fr.Proto)
	require.EqualValues(t, 120000, fr.Bytes)
	require.EqualValues(t, 100, fr.Packets)
	require.Equal(t, exp, fr.Exporter)
}

func TestFlowFromFieldsV6AndWideCounters(t *testing.T) {
	v6 := netip.MustParseAddr("2001:db8::9").As16()
	v6d := netip.MustParseAddr("2001:db8::1").As16()
	fields := []nf.DataField{
		{Type: 27, Value: v6[:]},
		{Type: 28, Value: v6d[:]},
		{Type: 1, Value: []byte{0, 0, 0, 0, 0, 0, 0x27, 0x10}}, // 8-byte octets = 10000
		{Type: 2, Value: []byte{0, 0, 0, 5}},                   // 4-byte packets = 5
	}
	fr, ok := flowFromFields(fields, netip.Addr{}, time.Now())
	require.True(t, ok)
	require.Equal(t, "2001:db8::9", fr.Src.String())
	require.EqualValues(t, 10000, fr.Bytes)
	require.EqualValues(t, 5, fr.Packets)
}

func TestFlowFromFieldsDropsNoAddr(t *testing.T) {
	_, ok := flowFromFields([]nf.DataField{{Type: 4, Value: []byte{6}}}, netip.Addr{}, time.Now())
	require.False(t, ok)
}

func TestParserV5StillWorks(t *testing.T) {
	data := buildV5(t, rec{"10.0.0.9", "10.0.0.1", 50000, 443, 6, 100, 120000})
	recs := NewParser().Parse(data, netip.MustParseAddr("10.0.0.254"), time.Now())
	require.Len(t, recs, 1)
	require.Equal(t, "10.0.0.9", recs[0].Src.String())
}

func TestParseV9Combined(t *testing.T) {
	pkt := append(append(v9Header(2), v9TemplateFS()...), v9DataFS("10.0.0.9", "10.0.0.1", 50000, 443, 6, 120000, 100)...)
	recs := NewParser().Parse(pkt, netip.MustParseAddr("10.0.0.254"), time.Now())
	require.Len(t, recs, 1)
	require.Equal(t, "10.0.0.9", recs[0].Src.String())
	require.Equal(t, "10.0.0.1", recs[0].Dst.String())
	require.EqualValues(t, 443, recs[0].DstPort)
	require.EqualValues(t, 120000, recs[0].Bytes)
}

func TestParseV9SeparateTemplate(t *testing.T) {
	p := NewParser()
	exp := netip.MustParseAddr("10.0.0.254")
	// data before template → no records (template unknown)
	require.Empty(t, p.Parse(append(v9Header(1), v9DataFS("10.0.0.9", "10.0.0.1", 50000, 443, 6, 120000, 100)...), exp, time.Now()))
	// template packet → learns it, 0 records
	require.Empty(t, p.Parse(append(v9Header(1), v9TemplateFS()...), exp, time.Now()))
	// now data decodes from the cached template
	recs := p.Parse(append(v9Header(1), v9DataFS("10.0.0.9", "10.0.0.1", 50000, 443, 6, 120000, 100)...), exp, time.Now())
	require.Len(t, recs, 1)
	require.Equal(t, "10.0.0.1", recs[0].Dst.String())
}

func TestParseGarbageTemplated(t *testing.T) {
	g := append(v9Header(1), []byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02}...)
	require.Empty(t, NewParser().Parse(g, netip.MustParseAddr("10.0.0.254"), time.Now())) // no panic
}
