package fwconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic Cisco ASA running-config fragment: two nameif interfaces (one
// outside at security-level 0), a name alias, a network object + object-group,
// a service object, a two-ACE access-list bound inbound on outside, and a
// tunnel-group pre-shared-key the parser must ignore.
const asaCfg = `interface GigabitEthernet0/0
 nameif outside
 security-level 0
 ip address 203.0.113.1 255.255.255.0
!
interface GigabitEthernet0/1
 nameif inside
 security-level 100
 ip address 10.0.10.1 255.255.255.0
!
name 10.0.20.10 web-server
object network web-host
 host 10.0.20.10
object network dmz-net
 subnet 10.0.30.0 255.255.255.0
object-group network internal-nets
 network-object 10.0.10.0 255.255.255.0
 network-object object dmz-net
object service https-svc
 service tcp destination eq 443
access-list OUTSIDE-IN extended permit tcp any object web-host eq 443
access-list OUTSIDE-IN extended deny ip any any
access-group OUTSIDE-IN in interface outside
tunnel-group 203.0.113.9 ipsec-attributes
 ikev2 remote-authentication pre-shared-key SECRETKEY99`

func asa(t *testing.T, cfg string) Analysis {
	t.Helper()
	a, err := analyzeAsa(cfg)
	require.NoError(t, err)
	require.Len(t, a, 1)
	return a[0]
}

func TestAsaInterfacesZones(t *testing.T) {
	a := asa(t, asaCfg)
	require.Len(t, a.VLANs, 2)
	byName := map[string]VLAN{}
	for _, v := range a.VLANs {
		byName[v.Name] = v
	}
	require.Equal(t, "203.0.113.0/24", byName["outside"].Subnet)
	require.Equal(t, "GigabitEthernet0/0", byName["outside"].Interface)
	require.Equal(t, "10.0.10.0/24", byName["inside"].Subnet)
	require.Equal(t, "outside", byName["outside"].Zone)

	zones := map[string][]string{}
	for _, z := range a.Zones {
		zones[z.Name] = z.Interfaces
	}
	require.Equal(t, []string{"outside"}, zones["outside"])
	require.Equal(t, []string{"inside"}, zones["inside"])
}

func TestAsaObjects(t *testing.T) {
	a := asa(t, asaCfg)
	addr := map[string]Address{}
	for _, x := range a.Addresses {
		addr[x.Name] = x
	}
	require.Equal(t, "subnet", addr["web-host"].Kind)
	require.Equal(t, "10.0.20.10/32", addr["web-host"].Value)
	require.Equal(t, "subnet", addr["dmz-net"].Kind)
	require.Equal(t, "10.0.30.0/24", addr["dmz-net"].Value)
	require.Equal(t, "group", addr["internal-nets"].Kind)

	svc := map[string]Service{}
	for _, s := range a.Services {
		svc[s.Name] = s
	}
	require.Equal(t, "TCP", svc["https-svc"].Proto)
	require.Equal(t, "443", svc["https-svc"].Ports)
}

func TestAsaAclToFlows(t *testing.T) {
	a := asa(t, asaCfg)
	require.Len(t, a.Policies, 2) // two ACEs under OUTSIDE-IN
	for _, p := range a.Policies {
		require.Equal(t, "OUTSIDE-IN", p.Name)
		require.Equal(t, []string{"outside"}, p.SrcIntf) // access-group ... in interface outside
		require.NotZero(t, p.ID)
	}
	actions := map[string]bool{}
	for _, p := range a.Policies {
		actions[p.Action] = true
	}
	require.True(t, actions["accept"])
	require.True(t, actions["deny"])

	// a permit flow from the bound interface exists, carrying the service
	var allow *Flow
	for i := range a.Flows {
		if a.Flows[i].Action == "allow" {
			allow = &a.Flows[i]
		}
	}
	require.NotNil(t, allow)
	require.Equal(t, "outside", allow.Src)
	require.Contains(t, allow.Services, "tcp/443")

	// ASA's interface-binding limitation is surfaced as a Note
	joined := ""
	for _, n := range a.Notes {
		joined += n + "\n"
	}
	require.Contains(t, joined, "ASA")
}

func TestAsaDanglingAccessGroup(t *testing.T) {
	cfg := `interface GigabitEthernet0/1
 nameif inside
 ip address 10.0.10.1 255.255.255.0
access-list GHOST extended permit ip any any
access-group REAL in interface inside`
	a, err := analyzeAsa(cfg)
	require.NoError(t, err)
	require.Len(t, a, 1)
	// GHOST's ACE is kept even though nothing binds it (empty SrcIntf → Internet)
	require.Len(t, a[0].Policies, 1)
	require.Empty(t, a[0].Policies[0].SrcIntf)
	// no panic on an access-group naming a nonexistent ACL
}

// Standard ACLs aren't interface traffic policies; they must not appear as a
// policy with a bogus (inverted) action and garbage protocol.
func TestAsaStandardAclSkipped(t *testing.T) {
	cfg := `access-list SPLIT standard permit 10.0.0.0 255.0.0.0
access-list OUT extended permit tcp any any eq 80
access-group OUT in interface inside
interface GigabitEthernet0/0
 nameif inside
 ip address 10.0.0.1 255.255.255.0`
	a, err := analyzeAsa(cfg)
	require.NoError(t, err)
	require.Len(t, a[0].Policies, 1) // only the extended ACE
	require.Equal(t, "accept", a[0].Policies[0].Action)
	require.Equal(t, []string{"tcp/80"}, a[0].Policies[0].Services)
}

// An ACE with a source-port operator must not corrupt the destination or lose
// the destination service.
func TestAsaSourcePortAce(t *testing.T) {
	cfg := `access-list OUT extended permit tcp host 1.1.1.1 eq 1024 any eq 443
access-group OUT in interface outside
interface GigabitEthernet0/0
 nameif outside
 ip address 203.0.113.1 255.255.255.0`
	a, err := analyzeAsa(cfg)
	require.NoError(t, err)
	require.Len(t, a[0].Policies, 1)
	p := a[0].Policies[0]
	require.Equal(t, []string{"1.1.1.1/32"}, p.SrcAddr)
	require.Equal(t, []string{"any"}, p.DstAddr)
	require.Equal(t, []string{"tcp/443"}, p.Services)
}

// A nested group-object inside an object-group must be included as a member.
func TestAsaNestedObjectGroup(t *testing.T) {
	cfg := `object-group network inner
 network-object host 10.0.0.5
object-group network outer
 group-object inner
 network-object 10.0.1.0 255.255.255.0`
	a, err := analyzeAsa(cfg)
	require.NoError(t, err)
	outer := map[string]Address{}
	for _, x := range a[0].Addresses {
		outer[x.Name] = x
	}
	require.Contains(t, outer["outer"].Value, "inner")
	require.Contains(t, outer["outer"].Value, "10.0.1.0/24")
}

// An object service with a port range (not eq) must still capture the ports.
func TestAsaServiceRange(t *testing.T) {
	cfg := `object service bigports
 service tcp destination range 1000 2000`
	a, err := analyzeAsa(cfg)
	require.NoError(t, err)
	require.Len(t, a[0].Services, 1)
	require.Equal(t, "1000-2000", a[0].Services[0].Ports)
}

func TestAsaSecretNotSurfaced(t *testing.T) {
	a := asa(t, asaCfg)
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "SECRETKEY99")
}
