package ndp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// raBody builds a realistic Router Advertisement body (from Cur Hop Limit on):
// O flag set, router lifetime 1800s, a source MAC, a SLAAC prefix, MTU 1500,
// an RDNSS server, and a DNSSL search domain.
func raBody() []byte {
	b := []byte{64, 0x40, 0x07, 0x08, 0, 0, 0, 0, 0, 0, 0, 0} // hop 64, O flag, lifetime 1800

	b = append(b, 1, 1, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff) // source LL addr

	po := []byte{3, 4, 64, 0xC0}         // prefix info: /64, L+A
	po = append(po, 0, 0x01, 0x51, 0x80) // valid 86400
	po = append(po, 0, 0, 0x38, 0x40)    // preferred 14400
	po = append(po, 0, 0, 0, 0)          // reserved
	po = append(po, net.ParseIP("2001:db8::").To16()...)
	b = append(b, po...)

	b = append(b, 5, 1, 0, 0, 0, 0, 0x05, 0xDC) // MTU 1500

	rd := []byte{25, 3, 0, 0, 0, 0, 0, 30} // RDNSS, lifetime 30
	rd = append(rd, net.ParseIP("2606:4700:4700::1111").To16()...)
	b = append(b, rd...)

	ds := []byte{31, 0, 0, 0, 0, 0, 0, 30} // DNSSL, lifetime 30 (len filled below)
	ds = append(ds, 7)
	ds = append(ds, []byte("example")...)
	ds = append(ds, 3)
	ds = append(ds, []byte("com")...)
	ds = append(ds, 0)
	for len(ds)%8 != 0 {
		ds = append(ds, 0)
	}
	ds[1] = byte(len(ds) / 8)
	b = append(b, ds...)
	return b
}

func TestParseRA(t *testing.T) {
	ra, err := parseRA(raBody(), "fe80::1", "en0")
	require.NoError(t, err)

	require.False(t, ra.Managed)
	require.True(t, ra.OtherConfig)
	require.Equal(t, uint16(1800), ra.RouterLifetime)
	require.Equal(t, uint8(64), ra.HopLimit)
	require.Equal(t, "aa:bb:cc:dd:ee:ff", ra.SourceMAC)

	require.Len(t, ra.Prefixes, 1)
	require.Equal(t, "2001:db8::/64", ra.Prefixes[0].Prefix)
	require.True(t, ra.Prefixes[0].OnLink)
	require.True(t, ra.Prefixes[0].Autonomous)
	require.Equal(t, uint32(86400), ra.Prefixes[0].ValidLifetime)
	require.Equal(t, uint32(14400), ra.Prefixes[0].PreferredLifetime)

	require.Equal(t, uint32(1500), ra.MTU)
	require.Equal(t, []string{"2606:4700:4700::1111"}, ra.DNS)
	require.Equal(t, []string{"example.com"}, ra.DNSSearch)
}

func TestParseRARejectsShort(t *testing.T) {
	_, err := parseRA([]byte{1, 2, 3}, "fe80::1", "en0")
	require.Error(t, err)
}

func TestParseRAStopsOnBadOptionLength(t *testing.T) {
	// A zero option length must not loop forever or overrun.
	body := append([]byte{64, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 3, 0, 0, 0)
	ra, err := parseRA(body, "fe80::1", "en0")
	require.NoError(t, err)
	require.Empty(t, ra.Prefixes)
}
