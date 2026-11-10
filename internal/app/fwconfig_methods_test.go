package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const fgSample = `config system interface
    edit "port1.30"
        set vlanid 30
        set ip 10.0.30.1 255.255.255.0
    next
end
config firewall policy
    edit 1
        set srcintf "port1.30"
        set dstintf "port1.20"
        set action accept
    next
end`

func TestAnalyzeFirewallEmpty(t *testing.T) {
	_, err := New().AnalyzeFirewall("", "fortigate")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "config")
}

func TestAnalyzeFirewallFortigate(t *testing.T) {
	a, err := New().AnalyzeFirewall(fgSample, "fortigate")
	require.NoError(t, err)
	require.Len(t, a, 1)
	require.Len(t, a[0].VLANs, 1)
	require.Equal(t, 30, a[0].VLANs[0].ID)
	require.NotEmpty(t, a[0].Flows)
}

func TestAnalyzeFirewallUnknownVendor(t *testing.T) {
	_, err := New().AnalyzeFirewall("config system global\nend", "cisco")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "support")
}

func TestAnalyzeFirewallDefaultVendor(t *testing.T) {
	a, err := New().AnalyzeFirewall(fgSample, "")
	require.NoError(t, err)
	require.Len(t, a, 1)
}
