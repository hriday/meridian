package wwn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNAA2Cisco(t *testing.T) {
	// 0x2x → NAA 2; OUI is bytes 2-4 (00:25:b5 = Cisco).
	r, err := Parse("20:00:00:25:b5:00:00:01")
	require.NoError(t, err)
	require.Equal(t, "2", r.NAA)
	require.Equal(t, "IEEE Extended (NAA 2)", r.NAADesc)
	require.Equal(t, "0025b5", r.OUIBare)
	require.Equal(t, "00:25:b5", r.OUI)
	require.Equal(t, "000001", r.VSID)
	require.Equal(t, "20:00:00:25:b5:00:00:01", r.Canonical)
	require.Equal(t, "2000.0025.b500.0001", r.Dotted)
}

func TestParseNAA5NetApp(t *testing.T) {
	// 0x5x → NAA 5; OUI is the 6 nibbles after the NAA nibble (00:a0:98 = NetApp).
	r, err := Parse("50:0a:09:81:88:4b:2f:a0")
	require.NoError(t, err)
	require.Equal(t, "5", r.NAA)
	require.Equal(t, "00a098", r.OUIBare)
}

func TestParseNAA1Emulex(t *testing.T) {
	// 0x1x → NAA 1; OUI is bytes 2-4 (00:00:c9 = Emulex). Bare-hex input accepted.
	r, err := Parse("10000000c9abcdef")
	require.NoError(t, err)
	require.Equal(t, "1", r.NAA)
	require.Equal(t, "0000c9", r.OUIBare)
	require.Equal(t, "abcdef", r.VSID)
}

func TestParseAcceptsAnySpelling(t *testing.T) {
	for _, in := range []string{
		"2000.0025.b500.0001",
		"20-00-00-25-b5-00-00-01",
		"20 00 00 25 b5 00 00 01",
		"20000025B5000001",
	} {
		r, err := Parse(in)
		require.NoError(t, err, in)
		require.Equal(t, "20:00:00:25:b5:00:00:01", r.Canonical, in)
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	_, err := Parse("20:00:00") // too short
	require.Error(t, err)
	_, err = Parse("zz:00:00:25:b5:00:00:01") // non-hex
	require.Error(t, err)
}

func TestParseNAA6Is128Bit(t *testing.T) {
	r, err := Parse("60000025b5000001deadbeefcafe0000")
	require.NoError(t, err)
	require.Equal(t, "6", r.NAA)
	require.Equal(t, 128, r.Bits)
}
