package dnsbl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReverseNameIPv4(t *testing.T) {
	got, err := reverseName("1.2.3.4", "zen.spamhaus.org")
	require.NoError(t, err)
	require.Equal(t, "4.3.2.1.zen.spamhaus.org", got)
}

func TestReverseNameIPv6(t *testing.T) {
	// 2001:db8::1 expands to 2001:0db8:0000:0000:0000:0000:0000:0001; its 32
	// nibbles reversed and dotted, then the zone.
	got, err := reverseName("2001:db8::1", "z")
	require.NoError(t, err)
	require.Equal(t, "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.z", got)
}

func TestReverseNameRejectsNonIP(t *testing.T) {
	_, err := reverseName("example.com", "z")
	require.Error(t, err)
}

func TestClassifyListed(t *testing.T) {
	s, _ := classify([]string{"127.0.0.2"})
	require.Equal(t, StatusListed, s)
	// A listing code other than .2 (Spamhaus PBL is .10/.11) must still read as listed.
	s, _ = classify([]string{"127.0.0.10"})
	require.Equal(t, StatusListed, s)
}

func TestClassifyClean(t *testing.T) {
	s, _ := classify(nil)
	require.Equal(t, StatusClean, s)
}

func TestClassifyBlocked(t *testing.T) {
	for _, code := range []string{"127.255.255.252", "127.255.255.254", "127.255.255.255"} {
		s, note := classify([]string{code})
		require.Equal(t, StatusBlocked, s, code)
		require.NotEmpty(t, note, code)
	}
	// The public-resolver code's note should point at the resolver.
	_, note := classify([]string{"127.255.255.254"})
	require.Contains(t, note, "resolver")
}

func TestClassifyErrorOnOddAnswer(t *testing.T) {
	// An answer outside 127/8 is not a valid DNSBL response.
	s, _ := classify([]string{"10.0.0.1"})
	require.Equal(t, StatusError, s)
}

func TestDefaultZonesNonEmpty(t *testing.T) {
	zones := DefaultZones()
	require.GreaterOrEqual(t, len(zones), 3)
	var hasSpamhaus bool
	for _, z := range zones {
		require.NotEmpty(t, z.Host)
		require.NotEmpty(t, z.Name)
		if z.Host == "zen.spamhaus.org" {
			hasSpamhaus = true
		}
	}
	require.True(t, hasSpamhaus)
}
