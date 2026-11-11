package snmp

import (
	"encoding/json"
	"net/netip"
	"testing"

	g "github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/require"
)

var trapSrc = netip.MustParseAddr("10.0.0.3")

// encodeV2cTrap builds a v2c trap datagram: sysUpTime.0, snmpTrapOID.0=linkDown,
// and one data varbind, under the given community.
func encodeV2cTrap(t *testing.T, community string) []byte {
	t.Helper()
	gs := &g.GoSNMP{Version: g.Version2c, Community: community}
	pdus := []g.SnmpPDU{
		{Name: "1.3.6.1.2.1.1.3.0", Type: g.TimeTicks, Value: uint32(12345)},
		{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier, Value: "1.3.6.1.6.3.1.1.5.3"},
		{Name: "1.3.6.1.2.1.2.2.1.1.2", Type: g.Integer, Value: 2},
	}
	data, err := gs.SnmpEncodePacket(g.SNMPv2Trap, pdus, 0, 0)
	require.NoError(t, err)
	return data
}

func TestParseTrapV2c(t *testing.T) {
	m, ok := ParseTrap(encodeV2cTrap(t, "public"), trapSrc)
	require.True(t, ok)
	require.Equal(t, "v2c", m.Version)
	require.Equal(t, "1.3.6.1.6.3.1.1.5.3", m.TrapOID)
	require.Equal(t, "linkDown", m.TrapName)
	require.Equal(t, "10.0.0.3", m.Source)
	require.NotEmpty(t, m.Varbinds)
}

func TestParseTrapNoCommunity(t *testing.T) {
	m, ok := ParseTrap(encodeV2cTrap(t, "s3cr3tCommunity"), trapSrc)
	require.True(t, ok)
	b, _ := json.Marshal(m)
	require.NotContains(t, string(b), "s3cr3tCommunity")
}

func TestParseTrapGarbage(t *testing.T) {
	_, ok := ParseTrap([]byte{0x00, 0x01, 0x02, 0x03, 0xff}, trapSrc)
	require.False(t, ok)
	_, ok = ParseTrap(nil, trapSrc)
	require.False(t, ok)
}
