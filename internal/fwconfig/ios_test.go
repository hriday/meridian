package fwconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic Cisco IOS-XE running-config excerpt (object-group, named extended
// ACL bound to an interface, a dangerous any→any permit).
const iosXEConfig = `!
hostname EDGE-RTR
!
object-group network WEB-SERVERS
 host 10.1.1.10
 host 10.1.1.11
 10.1.2.0 255.255.255.0
!
interface GigabitEthernet0/0
 description Uplink to ISP
 ip address 203.0.113.2 255.255.255.0
 ip access-group OUTSIDE-IN in
!
interface GigabitEthernet0/1
 description Inside LAN
 ip address 10.1.1.1 255.255.255.0
!
ip access-list extended OUTSIDE-IN
 permit tcp any object-group WEB-SERVERS eq 443
 permit tcp any host 10.1.1.10 eq 22
 permit tcp any any eq 3389
 deny ip any any
!
`

func TestIOSXEParsesACLAndBinding(t *testing.T) {
	out, err := analyzeIOS(iosXEConfig)
	require.NoError(t, err)
	require.Len(t, out, 1)
	a := out[0]

	// Two L3 interfaces are segments.
	require.Equal(t, 2, a.Counts.VLANs)
	// The object-group becomes one address group with 3 members.
	var web *Address
	for i := range a.Addresses {
		if a.Addresses[i].Name == "WEB-SERVERS" {
			web = &a.Addresses[i]
		}
	}
	require.NotNil(t, web, "WEB-SERVERS object-group should be an address")
	require.Equal(t, "group", web.Kind)
	require.Contains(t, web.Value, "10.1.1.10/32")
	require.Contains(t, web.Value, "10.1.2.0/24")

	// Four ACEs → four policies, all named OUTSIDE-IN and bound to Gig0/0.
	require.Len(t, a.Policies, 4)
	for _, p := range a.Policies {
		require.Equal(t, "OUTSIDE-IN", p.Name)
		require.Equal(t, []string{"GigabitEthernet0/0"}, p.SrcIntf)
	}
	// The RDP-to-any permit is the exposed one.
	var rdp *Policy
	for i := range a.Policies {
		if len(a.Policies[i].Services) == 1 && a.Policies[i].Services[0] == "tcp/3389" {
			rdp = &a.Policies[i]
		}
	}
	require.NotNil(t, rdp)
	require.Equal(t, "any", rdp.SrcAddr[0])
	require.Equal(t, "any", rdp.DstAddr[0])
	require.Equal(t, "accept", rdp.Action)

	// Flows are derived from the bound interface.
	require.NotEmpty(t, a.Flows)
}

// NX-OS uses sequence-numbered ACEs and `ip access-list NAME` (no "extended").
const nxosConfig = `interface Ethernet1/1
  ip address 192.168.50.1/24
  ip access-group GUEST-IN in
!
ip access-list GUEST-IN
  10 permit tcp 192.168.50.0/24 any eq 80
  20 permit tcp 192.168.50.0/24 any eq 443
  30 deny ip any any
`

func TestNXOSSequencedACL(t *testing.T) {
	out, err := analyzeIOS(nxosConfig)
	require.NoError(t, err)
	a := out[0]
	require.Len(t, a.Policies, 3, "three sequenced ACEs")
	require.Equal(t, "192.168.50.0/24", a.Policies[0].SrcAddr[0])
	require.Equal(t, "tcp/80", a.Policies[0].Services[0])
	require.Equal(t, []string{"Ethernet1/1"}, a.Policies[0].SrcIntf)
}

func TestIOSFlatNumberedACL(t *testing.T) {
	cfg := "access-list 101 permit tcp any host 198.51.100.5 eq 25\naccess-list 101 deny ip any any\n"
	out, err := analyzeIOS(cfg)
	require.NoError(t, err)
	require.Len(t, out[0].Policies, 2)
	require.Equal(t, "101", out[0].Policies[0].Name)
	require.Equal(t, "198.51.100.5/32", out[0].Policies[0].DstAddr[0])
}

func TestIOSAnalyzeViaDispatch(t *testing.T) {
	out, err := Analyze(iosXEConfig, "ios-xe")
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.True(t, strings.Contains(out[0].Notes[0], "Cisco IOS"))
}

func TestDetectVendor(t *testing.T) {
	cases := map[string]string{
		iosXEConfig:    "ios",
		nxosConfig:     "ios",
		junosSetConfig: "junos",
		"config firewall policy\n  edit 1\n":                    "fortigate",
		"ASA Version 9.8\nnameif outside\n":                     "asa",
		"set security policies from-zone trust to-zone untrust": "srx",
		"set rulebase security rules r1 action allow":           "panos",
	}
	for cfg, want := range cases {
		if got := DetectVendor(cfg); got != want {
			t.Errorf("DetectVendor(%.30q) = %q, want %q", cfg, got, want)
		}
	}
}
