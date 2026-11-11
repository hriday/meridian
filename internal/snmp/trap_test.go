package snmp

import (
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"
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

func TestParseTrapRejectsNonTrap(t *testing.T) {
	// a GetRequest/Response/Set to the trap port must not be ingested as a trap.
	gs := &g.GoSNMP{Version: g.Version2c, Community: "public"}
	for _, pt := range []g.PDUType{g.GetRequest, g.GetResponse, g.SetRequest} {
		data, err := gs.SnmpEncodePacket(pt, []g.SnmpPDU{{Name: "1.3.6.1.2.1.1.1.0", Type: g.Null}}, 0, 0)
		require.NoError(t, err)
		_, ok := ParseTrap(data, trapSrc)
		require.False(t, ok, "PDUType %v must be rejected", pt)
	}
}

func TestParseTrapBoundsVarbinds(t *testing.T) {
	gs := &g.GoSNMP{Version: g.Version2c, Community: "public"}
	pdus := []g.SnmpPDU{
		{Name: "1.3.6.1.6.3.1.1.4.1.0", Type: g.ObjectIdentifier, Value: "1.3.6.1.6.3.1.1.5.3"},
		{Name: "1.3.6.1.4.1.9.1", Type: g.OctetString, Value: strings.Repeat("A", 5000)},
	}
	for i := 0; i < 300; i++ {
		pdus = append(pdus, g.SnmpPDU{Name: "1.3.6.1.4.1.9." + strconv.Itoa(i+2), Type: g.Integer, Value: i})
	}
	data, err := gs.SnmpEncodePacket(g.SNMPv2Trap, pdus, 0, 0)
	require.NoError(t, err)
	m, ok := ParseTrap(data, trapSrc)
	require.True(t, ok)
	require.Equal(t, "1.3.6.1.6.3.1.1.5.3", m.TrapOID) // OID extraction unaffected by the cap
	require.LessOrEqual(t, len(m.Varbinds), 64)
	for _, v := range m.Varbinds {
		require.LessOrEqual(t, len(v.Value), 512)
	}
}

func TestParseTrapGarbage(t *testing.T) {
	_, ok := ParseTrap([]byte{0x00, 0x01, 0x02, 0x03, 0xff}, trapSrc)
	require.False(t, ok)
	_, ok = ParseTrap(nil, trapSrc)
	require.False(t, ok)
}
