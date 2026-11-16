package ndp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDarwinNeighbors(t *testing.T) {
	// Real `ndp -an` shape, plus a router line (Flgs column "R").
	out := `Neighbor                                Linklayer Address  Netif Expire    St Flgs Prbs
fe80::c08:4587:a625:a9dc%en0            be:b6:72:3b:b0:ad    en0 permanent R
fe80::1%en0                             a4:b1:c2:d3:e4:f5    en0 23h59m58s  R  R
fe80::aa7d:1908:8a58:c3a%utun0          (incomplete)       utun0 permanent R`
	ns := parseDarwinNeighbors(out)
	require.Len(t, ns, 3)

	require.Equal(t, "fe80::c08:4587:a625:a9dc", ns[0].IP)
	require.Equal(t, "be:b6:72:3b:b0:ad", ns[0].MAC)
	require.Equal(t, "en0", ns[0].Interface)
	require.Equal(t, "reachable", ns[0].State)
	require.False(t, ns[0].Router)

	require.True(t, ns[1].Router, "Flgs R marks a router")

	require.Empty(t, ns[2].MAC, "(incomplete) is not a MAC")
	require.Equal(t, "utun0", ns[2].Interface)
}

func TestParseLinuxNeighbors(t *testing.T) {
	out := `fe80::1 dev eth0 lladdr aa:bb:cc:dd:ee:ff router REACHABLE
2001:db8::5 dev eth0 lladdr 11:22:33:44:55:66 STALE
fe80::9 dev eth0 FAILED`
	ns := parseLinuxNeighbors(out)
	require.Len(t, ns, 3)

	require.Equal(t, "fe80::1", ns[0].IP)
	require.Equal(t, "eth0", ns[0].Interface)
	require.Equal(t, "aa:bb:cc:dd:ee:ff", ns[0].MAC)
	require.True(t, ns[0].Router)
	require.Equal(t, "reachable", ns[0].State)

	require.Equal(t, "stale", ns[1].State)
	require.False(t, ns[1].Router)

	require.Empty(t, ns[2].MAC)
	require.Equal(t, "failed", ns[2].State)
}

func TestParseWindowsNeighbors(t *testing.T) {
	out := `Interface 12: Ethernet

Internet Address                              Physical Address   Type
--------------------------------------------  -----------------  -----------
fe80::1                                       aa-bb-cc-dd-ee-ff  Reachable (Router)
2001:db8::5                                   11-22-33-44-55-66  Stale`
	ns := parseWindowsNeighbors(out)
	require.Len(t, ns, 2)

	require.Equal(t, "fe80::1", ns[0].IP)
	require.Equal(t, "Ethernet", ns[0].Interface)
	require.Equal(t, "aa:bb:cc:dd:ee:ff", ns[0].MAC)
	require.True(t, ns[0].Router)
	require.Equal(t, "reachable", ns[0].State)

	require.False(t, ns[1].Router)
	require.Equal(t, "stale", ns[1].State)
}
