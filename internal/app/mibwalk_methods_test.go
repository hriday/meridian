package app

import (
	"fmt"
	"testing"

	"github.com/hriday/meridian/internal/snmp"
	"github.com/stretchr/testify/require"
)

type mibFake struct{ rows []snmp.PDU }

func (f mibFake) Get([]string) ([]snmp.PDU, error) { return nil, nil }
func (f mibFake) Walk(string) ([]snmp.PDU, error)  { return f.rows, nil }
func (f mibFake) Close() error                     { return nil }

func TestMIBWalkReturnsRows(t *testing.T) {
	setDial(t, mibFake{rows: []snmp.PDU{
		{OID: "1.3.6.1.2.1.1.1.0", Type: "OctetString", Value: "Cisco IOS"},
		{OID: "1.3.6.1.2.1.1.5.0", Type: "OctetString", Value: "core-sw-1"},
	}}, nil)
	rows, err := New().MIBWalk("10.0.0.1", secret, "1.3.6.1.2.1.1")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "core-sw-1", rows[1].Value)
	require.Equal(t, "OctetString", rows[1].Type)
}

func TestMIBWalkDialErrorScrubbed(t *testing.T) {
	setDial(t, nil, fmt.Errorf("x509 bad host, community=%s", secret))
	_, err := New().MIBWalk("10.0.0.1", secret, ".1.3.6.1.2.1.1")
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
}
