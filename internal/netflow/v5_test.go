package netflow

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rec struct {
	src, dst     string
	sport, dport uint16
	proto        uint8
	pkts, octets uint32
}

func buildV5(t *testing.T, recs ...rec) []byte {
	t.Helper()
	buf := make([]byte, 24+48*len(recs))
	binary.BigEndian.PutUint16(buf[0:2], 5)
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(recs)))
	for i, r := range recs {
		o := 24 + 48*i
		s := netip.MustParseAddr(r.src).As4()
		d := netip.MustParseAddr(r.dst).As4()
		copy(buf[o:o+4], s[:])
		copy(buf[o+4:o+8], d[:])
		binary.BigEndian.PutUint32(buf[o+16:o+20], r.pkts)
		binary.BigEndian.PutUint32(buf[o+20:o+24], r.octets)
		binary.BigEndian.PutUint16(buf[o+32:o+34], r.sport)
		binary.BigEndian.PutUint16(buf[o+34:o+36], r.dport)
		buf[o+38] = r.proto
	}
	return buf
}

func TestParseV5(t *testing.T) {
	data := buildV5(t,
		rec{"10.0.0.9", "10.0.0.1", 50000, 443, 6, 100, 120000},
		rec{"10.0.2.5", "8.8.8.8", 33000, 53, 17, 2, 240},
	)
	exp := netip.MustParseAddr("10.0.0.254")
	recs := ParseV5(data, exp, time.Now())
	require.Len(t, recs, 2)
	require.Equal(t, "10.0.0.9", recs[0].Src.String())
	require.Equal(t, "10.0.0.1", recs[0].Dst.String())
	require.EqualValues(t, 443, recs[0].DstPort)
	require.EqualValues(t, 6, recs[0].Proto)
	require.EqualValues(t, 100, recs[0].Packets)
	require.EqualValues(t, 120000, recs[0].Bytes)
	require.Equal(t, exp, recs[0].Exporter)
}

func TestParseV5Rejects(t *testing.T) {
	exp := netip.MustParseAddr("10.0.0.254")
	now := time.Now()
	require.Nil(t, ParseV5([]byte{0x00, 0x09}, exp, now)) // too short

	v9 := make([]byte, 48)
	binary.BigEndian.PutUint16(v9[0:2], 9)
	require.Nil(t, ParseV5(v9, exp, now)) // wrong version

	zero := make([]byte, 24)
	binary.BigEndian.PutUint16(zero[0:2], 5)
	require.Nil(t, ParseV5(zero, exp, now)) // count 0

	big := make([]byte, 24)
	binary.BigEndian.PutUint16(big[0:2], 5)
	binary.BigEndian.PutUint16(big[2:4], 99)
	require.Nil(t, ParseV5(big, exp, now)) // count claims 99 but body is short
}
