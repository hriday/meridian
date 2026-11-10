package wol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMagicPacketStructure(t *testing.T) {
	pkt, err := MagicPacket("01:02:03:04:05:06")
	require.NoError(t, err)
	require.Len(t, pkt, 102)
	require.Equal(t, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, pkt[0:6])
	mac := []byte{1, 2, 3, 4, 5, 6}
	for i := 0; i < 16; i++ {
		off := 6 + i*6
		require.Equalf(t, mac, pkt[off:off+6], "MAC repetition %d at offset %d", i, off)
	}
}

func TestMagicPacketAcceptsSpellings(t *testing.T) {
	ref, err := MagicPacket("01:02:03:04:05:06")
	require.NoError(t, err)
	for _, form := range []string{
		"01-02-03-04-05-06",
		"0102.0304.0506",
		"010203040506",   // bare hex
		"01:02:03:04:05:06",
	} {
		pkt, err := MagicPacket(form)
		require.NoError(t, err, form)
		require.True(t, bytes.Equal(ref, pkt), "form %q should yield the same packet", form)
	}
}

func TestMagicPacketRejects(t *testing.T) {
	for _, bad := range []string{"", "nope", "01:02:03:04:05", "01:02:03:04:05:06:07:08", "zz:02:03:04:05:06"} {
		_, err := MagicPacket(bad)
		require.Errorf(t, err, "expected error for %q", bad)
	}
}
