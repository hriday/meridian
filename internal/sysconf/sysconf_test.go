package sysconf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseResolvConf(t *testing.T) {
	in := `# generated
; a comment
nameserver 192.168.1.1
search lan
nameserver 2606:4700:4700::1111
`
	require.Equal(t, []string{"192.168.1.1", "2606:4700:4700::1111"}, parseResolvConf(in))
}

func TestParseScutilDNS(t *testing.T) {
	in := `DNS configuration
  resolver #1
    nameserver[0] : 192.168.1.1
    nameserver[1] : 1.1.1.1
  resolver #2
    nameserver[0] : 192.168.1.1
`
	require.Equal(t, []string{"192.168.1.1", "1.1.1.1"}, parseScutilDNS(in))
}

func TestParseSystemsetupNTP(t *testing.T) {
	require.Equal(t, "time.apple.com", parseSystemsetupNTP("Network Time Server: time.apple.com\n"))
	require.Equal(t, "", parseSystemsetupNTP("Network Time Server: (null)\n"))
}

func TestParseTimedatectl(t *testing.T) {
	require.Equal(t, "time.cloudflare.com", parseTimedatectl("time.cloudflare.com\n"))
}

func TestParseW32tm(t *testing.T) {
	require.Equal(t, "time.windows.com", parseW32tmSource("time.windows.com\r\n"))
}

func TestParseIpconfigDNS(t *testing.T) {
	in := "   DNS Servers . . . . . . . . . . . : 192.168.1.1\r\n" +
		"                                       8.8.8.8\r\n" +
		"   NetBIOS over Tcpip. . . . . . . . : Enabled\r\n"
	require.Equal(t, []string{"192.168.1.1", "8.8.8.8"}, parseIpconfigDNS(in))
}

func TestParseIpconfigDNSIPv6(t *testing.T) {
	in := "   DNS Servers . . . . . . . . . . . : 192.168.1.1\r\n" +
		"                                       fe80::1\r\n" +
		"                                       2606:4700:4700::1111\r\n" +
		"   NetBIOS over Tcpip. . . . . . . . : Enabled\r\n"
	require.Equal(t, []string{"192.168.1.1", "fe80::1", "2606:4700:4700::1111"}, parseIpconfigDNS(in))
}

func TestParseNtpConfServer(t *testing.T) {
	require.Equal(t, "pool.ntp.org", parseNtpConfServer("# cfg\nserver pool.ntp.org iburst\n"))
	require.Equal(t, "time.nist.gov", parseNtpConfServer("[Time]\nNTP=time.nist.gov\n"))
}

func TestParsersOnGarbage(t *testing.T) {
	for _, s := range []string{"", "random unrelated text\n", "   "} {
		require.Empty(t, parseResolvConf(s), s)
		require.Empty(t, parseScutilDNS(s), s)
		require.Empty(t, parseIpconfigDNS(s), s)
		require.Equal(t, "", parseSystemsetupNTP(s), s)
		require.Equal(t, "", parseNtpConfServer(s), s)
	}
}
