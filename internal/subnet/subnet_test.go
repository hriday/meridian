package subnet_test

import (
	"testing"

	"github.com/hriday/meridian/internal/subnet"
	"github.com/stretchr/testify/require"
)

func TestDescribeIPv4(t *testing.T) {
	got, err := subnet.Describe("10.0.0.5/24") // host bits set
	require.NoError(t, err)
	require.Equal(t, "10.0.0.0/24", got.Prefix)
	require.Equal(t, "10.0.0.255", got.Broadcast)
	require.Equal(t, "255.255.255.0", got.Netmask)
	require.Equal(t, "0.0.0.255", got.Wildcard)
	require.Equal(t, "256", got.TotalCount)
	require.Equal(t, "10.0.0.1", got.UsableFirst)
	require.Equal(t, "10.0.0.254", got.UsableLast)
	require.Equal(t, "254", got.UsableCount)
}

func TestDescribeV4CornerPrefixes(t *testing.T) {
	s32, err := subnet.Describe("192.0.2.7/32")
	require.NoError(t, err)
	require.Equal(t, "1", s32.TotalCount)
	require.Equal(t, "192.0.2.7", s32.UsableFirst)
	require.Equal(t, "192.0.2.7", s32.UsableLast)
	require.Equal(t, "", s32.Broadcast)

	s31, err := subnet.Describe("192.0.2.6/31") // RFC 3021: both usable
	require.NoError(t, err)
	require.Equal(t, "2", s31.TotalCount)
	require.Equal(t, "2", s31.UsableCount)
	require.Equal(t, "192.0.2.6", s31.UsableFirst)
	require.Equal(t, "192.0.2.7", s31.UsableLast)
}

func TestDescribeIPv6Magnitude(t *testing.T) {
	got, err := subnet.Describe("2606:4700::/32")
	require.NoError(t, err)
	require.Equal(t, 6, got.Version)
	require.Equal(t, "", got.Broadcast)
	require.Equal(t, "79228162514264337593543950336", got.TotalCount) // 2^96
	require.Equal(t, got.TotalCount, got.UsableCount)                 // v6: all usable
}

func TestDescribeMalformed(t *testing.T) {
	for _, in := range []string{"", "1.1.1.1", "nonsense", "10.0.0.0/33"} {
		_, err := subnet.Describe(in)
		require.Error(t, err, in)
	}
}

func TestSplit(t *testing.T) {
	out, err := subnet.Split("10.0.0.0/24", 26)
	require.NoError(t, err)
	require.Equal(t, []string{"10.0.0.0/26", "10.0.0.64/26", "10.0.0.128/26", "10.0.0.192/26"}, out)
	_, err = subnet.Split("10.0.0.0/24", 23) // newLen must be longer
	require.Error(t, err)
}

func TestSummarize(t *testing.T) {
	out, err := subnet.Summarize([]string{"10.0.0.0/25", "10.0.0.128/25"})
	require.NoError(t, err)
	require.Equal(t, []string{"10.0.0.0/24"}, out)
	_, err = subnet.Summarize([]string{"10.0.0.0/24", "2001:db8::/32"})
	require.Error(t, err) // mixed families
}

func TestSummarizeNumericOrder(t *testing.T) {
	// Non-adjacent blocks stay separate and must come back in numeric,
	// not lexical, order (9, 10, 20, 100 — not 10, 100, 20, 9).
	out, err := subnet.Summarize([]string{"10.0.0.0/8", "9.0.0.0/8", "100.0.0.0/8", "20.0.0.0/8"})
	require.NoError(t, err)
	require.Equal(t, []string{"9.0.0.0/8", "10.0.0.0/8", "20.0.0.0/8", "100.0.0.0/8"}, out)
}

func TestSummarizeOverlapping(t *testing.T) {
	// A /25 fully contained in a /24 collapses to the /24.
	out, err := subnet.Summarize([]string{"10.0.0.0/24", "10.0.0.128/25"})
	require.NoError(t, err)
	require.Equal(t, []string{"10.0.0.0/24"}, out)
}

func TestSplitRejectsHugeExpansion(t *testing.T) {
	// A /8 split to /32 would be ~16M subnets; reject rather than allocate.
	_, err := subnet.Split("10.0.0.0/8", 32)
	require.Error(t, err)
	// A split within the cap still works.
	out, err := subnet.Split("10.0.0.0/16", 24)
	require.NoError(t, err)
	require.Len(t, out, 256)
}
