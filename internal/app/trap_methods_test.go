package app

import (
	"net"
	"testing"
	"time"

	g "github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/require"
)

func encodeTrap(t *testing.T) []byte {
	t.Helper()
	gs := &g.GoSNMP{Version: g.Version2c, Community: "public"}
	data, err := gs.SnmpEncodePacket(g.SNMPv2Trap, []g.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.3.0", Type: g.TimeTicks, Value: uint32(1)},
		{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier, Value: "1.3.6.1.6.3.1.1.5.3"},
	}, 0, 0)
	require.NoError(t, err)
	return data
}

func TestTrapStartPollStop(t *testing.T) {
	a := New()
	require.NoError(t, a.TrapStart(0))
	defer a.TrapStop()
	port := a.TrapPoll(0).Port
	require.NotZero(t, port)

	c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	require.NoError(t, err)
	_, err = c.Write(encodeTrap(t))
	require.NoError(t, err)
	c.Close()

	require.Eventually(t, func() bool {
		for _, it := range a.TrapPoll(0).Items {
			if it.Val.TrapName == "linkDown" {
				return true
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond)

	a.TrapStop()
	require.False(t, a.TrapPoll(0).Running)
}

func TestTrapStartBindInUse(t *testing.T) {
	held, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)
	defer held.Close()
	require.Error(t, New().TrapStart(held.LocalAddr().(*net.UDPAddr).Port))
}
