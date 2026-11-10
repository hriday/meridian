package netif

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	ip, n, err := net.ParseCIDR(s)
	require.NoError(t, err)
	n.IP = ip
	return n
}

// setFakes installs fake seams returning the given interfaces and a per-index
// address map (index → addrs, or an error when present in errs).
func setFakes(t *testing.T, ifaces []net.Interface, addrs map[int][]net.Addr, errs map[int]error) {
	t.Helper()
	oldL, oldA := listInterfaces, interfaceAddrs
	listInterfaces = func() ([]net.Interface, error) { return ifaces, nil }
	interfaceAddrs = func(i net.Interface) ([]net.Addr, error) {
		if e, ok := errs[i.Index]; ok {
			return nil, e
		}
		return addrs[i.Index], nil
	}
	t.Cleanup(func() { listInterfaces, interfaceAddrs = oldL, oldA })
}

func TestListFormatsInterface(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	ifaces := []net.Interface{{
		Index: 2, Name: "en0", MTU: 1500, HardwareAddr: mac,
		Flags: net.FlagUp | net.FlagBroadcast | net.FlagMulticast | net.FlagRunning,
	}}
	addrs := map[int][]net.Addr{2: {mustCIDR(t, "192.168.1.20/24"), mustCIDR(t, "fe80::1/64")}}
	setFakes(t, ifaces, addrs, nil)

	got, err := List()
	require.NoError(t, err)
	require.Len(t, got, 1)
	i := got[0]
	require.Equal(t, "en0", i.Name)
	require.Equal(t, "aa:bb:cc:dd:ee:ff", i.MAC)
	require.Equal(t, 1500, i.MTU)
	require.True(t, i.Up)
	require.False(t, i.Loopback)
	require.Subset(t, i.Flags, []string{"up", "broadcast", "multicast"})
	require.Equal(t, []string{"192.168.1.20/24"}, i.IPv4)
	require.Equal(t, []string{"fe80::1/64"}, i.IPv6)
}

func TestListEmptyMAC(t *testing.T) {
	ifaces := []net.Interface{{Index: 1, Name: "lo0", MTU: 16384, Flags: net.FlagUp | net.FlagLoopback}}
	setFakes(t, ifaces, map[int][]net.Addr{1: {mustCIDR(t, "127.0.0.1/8")}}, nil)
	got, err := List()
	require.NoError(t, err)
	require.Equal(t, "", got[0].MAC)
	require.True(t, got[0].Loopback)
	require.Contains(t, got[0].Flags, "loopback")
}

func TestListAddrErrorStillListed(t *testing.T) {
	ifaces := []net.Interface{{Index: 3, Name: "utun0", MTU: 1400, Flags: net.FlagUp}}
	setFakes(t, ifaces, nil, map[int]error{3: errors.New("boom")})
	got, err := List()
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "utun0", got[0].Name)
	require.Empty(t, got[0].IPv4)
	require.Empty(t, got[0].IPv6)
}

func TestListSkipsNonIPNet(t *testing.T) {
	ifaces := []net.Interface{{Index: 1, Name: "en0", Flags: net.FlagUp}}
	addrs := map[int][]net.Addr{1: {&net.IPAddr{IP: net.ParseIP("10.0.0.1")}, mustCIDR(t, "10.0.0.2/24")}}
	setFakes(t, ifaces, addrs, nil)
	got, err := List()
	require.NoError(t, err)
	require.Equal(t, []string{"10.0.0.2/24"}, got[0].IPv4) // IPAddr skipped, no panic
}

func TestListSorted(t *testing.T) {
	ifaces := []net.Interface{{Index: 5, Name: "b"}, {Index: 1, Name: "a"}}
	setFakes(t, ifaces, nil, nil)
	got, err := List()
	require.NoError(t, err)
	require.Equal(t, 1, got[0].Index)
	require.Equal(t, 5, got[1].Index)
}

func TestListRealHostSmoke(t *testing.T) {
	got, err := List()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(got), 1) // loopback always exists
}
