package snmp

import (
	"fmt"
	"testing"

	g "github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/require"
)

func TestTypeName(t *testing.T) {
	require.Equal(t, "Counter32", typeName(g.Counter32))
	require.Equal(t, "TimeTicks", typeName(g.TimeTicks))
	require.Equal(t, "OctetString", typeName(g.OctetString))
	require.Equal(t, "Counter64", typeName(g.Counter64))
	require.Equal(t, "—", typeName(g.Boolean)) // unmapped → dash
}

func TestMIBWalkTypesAndValues(t *testing.T) {
	root := "1.3.6.1.2.1.1"
	c := &fakeClient{walk: map[string][]PDU{
		root: {
			{OID: root + ".1.0", Type: "OctetString", Value: "core-sw-1"},
			{OID: root + ".3.0", Type: "TimeTicks", Value: uint(1850)},
			{OID: root + ".7.0", Type: "Counter32", Value: uint(99)},
		},
	}}
	rows, err := MIBWalk(c, root, 100)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, MIBRow{OID: root + ".1.0", Type: "OctetString", Value: "core-sw-1"}, rows[0])
	require.Equal(t, "TimeTicks", rows[1].Type)
	require.Equal(t, "1850", rows[1].Value)
	require.Equal(t, "99", rows[2].Value)
}

func TestMIBWalkLeafGetFallback(t *testing.T) {
	oid := "1.3.6.1.2.1.1.5.0"
	c := &fakeClient{walk: map[string][]PDU{}, get: map[string]any{oid: "core-sw-1"}}
	rows, err := MIBWalk(c, oid, 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, oid, rows[0].OID)
	require.Equal(t, "core-sw-1", rows[0].Value)
}

func TestMIBWalkHexNonPrintable(t *testing.T) {
	root := "1.3.6.1.2.1.2.2.1.6.1" // ifPhysAddress-like raw bytes
	c := &fakeClient{walk: map[string][]PDU{
		root: {{OID: root, Type: "OctetString", Value: string([]byte{0x00, 0xde, 0xad, 0xbe})}},
	}}
	rows, err := MIBWalk(c, root, 100)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "00deadbe", rows[0].Value)
}

func TestMIBWalkKeepsPrintableWithNewline(t *testing.T) {
	root := "1.3.6.1.2.1.1.1.0"
	c := &fakeClient{walk: map[string][]PDU{
		root: {{OID: root, Type: "OctetString", Value: "Cisco IOS\nVersion 17.9"}},
	}}
	rows, err := MIBWalk(c, root, 100)
	require.NoError(t, err)
	require.Equal(t, "Cisco IOS\nVersion 17.9", rows[0].Value) // text w/ newline stays text
}

func TestMIBWalkCaps(t *testing.T) {
	root := "1.3.6.1.2.1"
	var pdus []PDU
	for i := 0; i < 10; i++ {
		pdus = append(pdus, PDU{OID: fmt.Sprintf("%s.%d", root, i), Type: "Integer", Value: i})
	}
	c := &fakeClient{walk: map[string][]PDU{root: pdus}}
	rows, err := MIBWalk(c, root, 4)
	require.NoError(t, err)
	require.Len(t, rows, 4)
}
