package fwconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic PAN-OS set-format export: interfaces (one parent + one tagged
// subinterface), two zones, address + address-group + range, service +
// service-group, two security rules built one-leaf-per-line (as real `set`
// output is), one with App-ID and one disabled, plus an IKE pre-shared-key line
// the parser must ignore.
const panosCfg = `set network interface ethernet ethernet1/1 layer3 ip 203.0.113.1/24
set network interface ethernet ethernet1/2 layer3 units ethernet1/2.20 ip 10.0.20.1/24
set network interface ethernet ethernet1/2 layer3 units ethernet1/2.20 tag 20
set zone untrust network layer3 ethernet1/1
set zone trust network layer3 ethernet1/2.20
set address H-web ip-netmask 10.0.20.10/32
set address Net-dmz ip-netmask 10.0.30.0/24
set address R-pool ip-range 10.0.40.10-10.0.40.20
set address-group G-servers static [ H-web Net-dmz ]
set service tcp-443 protocol tcp port 443
set service tcp-8080 protocol tcp port 8080
set service-group G-web members [ tcp-443 tcp-8080 ]
set rulebase security rules "Allow web" from trust
set rulebase security rules "Allow web" to untrust
set rulebase security rules "Allow web" source any
set rulebase security rules "Allow web" destination H-web
set rulebase security rules "Allow web" application ssl
set rulebase security rules "Allow web" service tcp-443
set rulebase security rules "Allow web" action allow
set rulebase security rules "Block old" from trust
set rulebase security rules "Block old" to untrust
set rulebase security rules "Block old" source any
set rulebase security rules "Block old" destination any
set rulebase security rules "Block old" application any
set rulebase security rules "Block old" service any
set rulebase security rules "Block old" action deny
set rulebase security rules "Block old" disabled yes
set network ike gateway GW1 authentication pre-shared-key authentication-value SUPERSECRETPSK123`

func panos(t *testing.T, cfg string) Analysis {
	t.Helper()
	a, err := analyzePanos(cfg)
	require.NoError(t, err)
	require.Len(t, a, 1)
	return a[0]
}

func TestPanosSegmentsZones(t *testing.T) {
	a := panos(t, panosCfg)
	require.Len(t, a.VLANs, 2)
	byName := map[string]VLAN{}
	for _, v := range a.VLANs {
		byName[v.Name] = v
	}
	require.Equal(t, "203.0.113.0/24", byName["ethernet1/1"].Subnet)
	require.Equal(t, "10.0.20.0/24", byName["ethernet1/2.20"].Subnet)
	require.Equal(t, 20, byName["ethernet1/2.20"].ID)
	require.Equal(t, "trust", byName["ethernet1/2.20"].Zone)
	require.Equal(t, "untrust", byName["ethernet1/1"].Zone)

	zones := map[string][]string{}
	for _, z := range a.Zones {
		zones[z.Name] = z.Interfaces
	}
	require.Equal(t, []string{"ethernet1/2.20"}, zones["trust"])
	require.Equal(t, []string{"ethernet1/1"}, zones["untrust"])
}

func TestPanosAddressesServices(t *testing.T) {
	a := panos(t, panosCfg)
	addr := map[string]Address{}
	for _, x := range a.Addresses {
		addr[x.Name] = x
	}
	require.Equal(t, "subnet", addr["H-web"].Kind)
	require.Equal(t, "10.0.20.10/32", addr["H-web"].Value)
	require.Equal(t, "iprange", addr["R-pool"].Kind)
	require.Equal(t, "group", addr["G-servers"].Kind)
	require.Contains(t, addr["G-servers"].Value, "H-web")
	require.Contains(t, addr["G-servers"].Value, "Net-dmz")

	svc := map[string]Service{}
	for _, s := range a.Services {
		svc[s.Name] = s
	}
	require.Equal(t, "TCP", svc["tcp-443"].Proto)
	require.Equal(t, "443", svc["tcp-443"].Ports)
	require.Equal(t, "group", svc["G-web"].Proto)
}

func TestPanosPoliciesAndFlows(t *testing.T) {
	a := panos(t, panosCfg)
	require.Len(t, a.Policies, 2)
	byName := map[string]Policy{}
	for _, p := range a.Policies {
		byName[p.Name] = p
		require.NotZero(t, p.ID) // synthetic sequential id assigned
	}
	require.Equal(t, "accept", byName["Allow web"].Action)
	require.Equal(t, "deny", byName["Block old"].Action)
	require.True(t, byName["Block old"].Disabled)

	// The App-ID rule must still be drawn as a flow (not suppressed to a Note),
	// with the application carried as a service-style label.
	require.NotEmpty(t, a.Flows)
	var allow *Flow
	for i := range a.Flows {
		if a.Flows[i].Action == "allow" {
			allow = &a.Flows[i]
		}
	}
	require.NotNil(t, allow, "Allow web should produce an allow flow")
	require.Contains(t, allow.Services, "tcp-443")
	require.Contains(t, allow.Services, "app:ssl")
	// disabled rule excluded
	for _, f := range a.Flows {
		require.NotContains(t, f.PolicyIDs, byName["Block old"].ID)
	}
	// untrust is treated as Internet-facing
	require.Equal(t, internetNode, allow.Dst)
}

func TestPanosSecretNotSurfaced(t *testing.T) {
	a := panos(t, panosCfg)
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "SUPERSECRETPSK123")
}

func TestPanosRejectsXML(t *testing.T) {
	_, err := analyzePanos("<config version=\"10.1.0\">\n  <devices/>\n</config>")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "set")
}
