package app

import (
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// v5one builds a one-record NetFlow v5 datagram.
func v5one(src, dst string, dport uint16, proto uint8, octets uint32) []byte {
	buf := make([]byte, 24+48)
	binary.BigEndian.PutUint16(buf[0:2], 5)
	binary.BigEndian.PutUint16(buf[2:4], 1)
	s := netip.MustParseAddr(src).As4()
	d := netip.MustParseAddr(dst).As4()
	copy(buf[24:28], s[:])
	copy(buf[28:32], d[:])
	binary.BigEndian.PutUint32(buf[24+20:24+24], octets)
	binary.BigEndian.PutUint16(buf[24+34:24+36], dport)
	buf[24+38] = proto
	return buf
}

func TestNetflowStartTopStop(t *testing.T) {
	a := New()
	require.NoError(t, a.NetflowStart(0))
	defer a.NetflowStop()
	port := a.NetflowTop(60, false).Port
	require.NotZero(t, port)

	c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	require.NoError(t, err)
	_, err = c.Write(v5one("10.0.0.9", "10.0.0.1", 443, 6, 120000))
	require.NoError(t, err)
	c.Close()

	require.Eventually(t, func() bool {
		v := a.NetflowTop(60, false)
		return len(v.Top) >= 1 && v.Top[0].DstIP == "10.0.0.1"
	}, 2*time.Second, 10*time.Millisecond)
	v := a.NetflowTop(60, false)
	require.GreaterOrEqual(t, v.Flows, 1)
	require.GreaterOrEqual(t, v.Exporters, 1)

	a.NetflowStop()
	require.False(t, a.NetflowTop(60, false).Running)
}

func TestNetflowNotTruncatedWhenSparse(t *testing.T) {
	// a freshly started, lightly-loaded collector covers little of a long window
	// but has evicted nothing — it must NOT report truncation.
	a := New()
	require.NoError(t, a.NetflowStart(0))
	defer a.NetflowStop()
	port := a.NetflowTop(300, false).Port
	c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	require.NoError(t, err)
	_, _ = c.Write(v5one("10.0.0.9", "10.0.0.1", 443, 6, 1000))
	c.Close()
	require.Eventually(t, func() bool { return a.NetflowTop(300, false).Flows >= 1 }, 2*time.Second, 10*time.Millisecond)
	require.False(t, a.NetflowTop(300, false).Truncated) // not saturated → no loss to report
}

func TestNetflowStartBindInUse(t *testing.T) {
	held, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer held.Close()
	require.Error(t, New().NetflowStart(held.LocalAddr().(*net.UDPAddr).Port))
}
