package ndp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDarwinRouters(t *testing.T) {
	out := `fe80::1%en0 if=en0, flags=IST, pref=medium, expire=29m50s
fe80::%utun0 if=utun0, flags=IST, pref=medium, expire=Never`
	rs := parseDarwinRouters(out)
	require.Len(t, rs, 1, "the bare fe80:: VPN-tunnel default route is filtered out")
	require.Equal(t, "fe80::1", rs[0].Address)
	require.Equal(t, "en0", rs[0].Interface)
	require.Equal(t, "medium", rs[0].Preference)
	require.Equal(t, "29m50s", rs[0].Expire)
}

func TestParseLinuxRouters(t *testing.T) {
	out := `default via fe80::1 dev eth0 proto ra metric 1024 expires 1512sec pref medium
fe80::/64 dev eth0 proto kernel metric 256`
	rs := parseLinuxRouters(out)
	require.Len(t, rs, 1)
	require.Equal(t, "fe80::1", rs[0].Address)
	require.Equal(t, "eth0", rs[0].Interface)
	require.Equal(t, "medium", rs[0].Preference)
	require.Equal(t, "1512sec", rs[0].Expire)
}

func TestParseWindowsRouters(t *testing.T) {
	out := `Publish  Type    Met  Prefix  Idx  Gateway/Interface Name
-------  ------  ---  ------  ---  ----------------------
No       Manual  256  ::/0    12   fe80::1
No       Manual  256  ::/0    5    Ethernet`
	rs := parseWindowsRouters(out)
	require.Len(t, rs, 1, "only the row with a real gateway address counts")
	require.Equal(t, "fe80::1", rs[0].Address)
}
