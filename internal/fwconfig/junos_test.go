package fwconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic non-SRX Junos config in set-format (`show configuration | display
// set`): a firewall filter with terms, bound to an interface unit as input.
const junosSetConfig = `set interfaces ge-0/0/0 unit 0 family inet address 203.0.113.2/24
set interfaces ge-0/0/0 unit 0 family inet filter input OUTSIDE-IN
set interfaces ge-0/0/1 unit 0 family inet address 10.1.1.1/24
set firewall family inet filter OUTSIDE-IN term allow-https from destination-address 10.1.1.10/32
set firewall family inet filter OUTSIDE-IN term allow-https from protocol tcp
set firewall family inet filter OUTSIDE-IN term allow-https from destination-port 443
set firewall family inet filter OUTSIDE-IN term allow-https then accept
set firewall family inet filter OUTSIDE-IN term allow-rdp from protocol tcp
set firewall family inet filter OUTSIDE-IN term allow-rdp from destination-port 3389
set firewall family inet filter OUTSIDE-IN term allow-rdp then accept
set firewall family inet filter OUTSIDE-IN term default-deny then discard
`

func TestJunosFilterTerms(t *testing.T) {
	out, err := analyzeJunos(junosSetConfig)
	require.NoError(t, err)
	require.Len(t, out, 1)
	a := out[0]

	// Two interface units are segments.
	require.Equal(t, 2, a.Counts.VLANs)

	// Three terms → three policies, all named OUTSIDE-IN, bound to ge-0/0/0.0.
	require.Len(t, a.Policies, 3)
	for _, p := range a.Policies {
		require.Equal(t, "OUTSIDE-IN", p.Name)
		require.Equal(t, []string{"ge-0/0/0.0"}, p.SrcIntf)
	}

	byPort := map[string]*Policy{}
	for i := range a.Policies {
		for _, s := range a.Policies[i].Services {
			byPort[s] = &a.Policies[i]
		}
	}
	require.Contains(t, byPort, "tcp/443")
	require.Equal(t, "accept", byPort["tcp/443"].Action)
	require.Equal(t, "10.1.1.10/32", byPort["tcp/443"].DstAddr[0])

	require.Contains(t, byPort, "tcp/3389")
	require.Equal(t, "accept", byPort["tcp/3389"].Action)

	// The default-deny term discards with no address → any/any deny.
	var deny *Policy
	for i := range a.Policies {
		if a.Policies[i].Action == "deny" {
			deny = &a.Policies[i]
		}
	}
	require.NotNil(t, deny)
	require.Equal(t, "any", deny.SrcAddr[0])
	require.Equal(t, "any", deny.DstAddr[0])

	require.NotEmpty(t, a.Flows)
}

func TestJunosViaDispatch(t *testing.T) {
	out, err := Analyze(junosSetConfig, "junos")
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Contains(t, out[0].Notes[0], "Junos firewall filters")
}

func TestJunosEmpty(t *testing.T) {
	if _, err := analyzeJunos("   "); err == nil {
		t.Fatal("empty config should error")
	}
}
