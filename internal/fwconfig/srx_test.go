package fwconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A realistic Juniper SRX `show configuration | display set` export: two
// interfaces (one tagged), trust/untrust zones, a global address-book with an
// address-set, two applications + an application-set, two from/to-zone policies
// (permit + deny), and an IKE pre-shared-key line the parser must ignore.
const srxCfg = `set interfaces ge-0/0/0 unit 0 family inet address 203.0.113.1/24
set interfaces ge-0/0/1 unit 20 vlan-id 20
set interfaces ge-0/0/1 unit 20 family inet address 10.0.20.1/24
set security zones security-zone untrust interfaces ge-0/0/0.0
set security zones security-zone trust interfaces ge-0/0/1.20
set security address-book global address web-host 10.0.20.10/32
set security address-book global address dmz-net 10.0.30.0/24
set security address-book global address-set servers address web-host
set security address-book global address-set servers address dmz-net
set applications application app-https protocol tcp destination-port 443
set applications application app-ssh protocol tcp destination-port 22
set applications application-set web-apps application app-https
set security policies from-zone trust to-zone untrust policy allow-web match source-address any
set security policies from-zone trust to-zone untrust policy allow-web match destination-address web-host
set security policies from-zone trust to-zone untrust policy allow-web match application app-https
set security policies from-zone trust to-zone untrust policy allow-web then permit
set security policies from-zone trust to-zone untrust policy block-ssh match source-address any
set security policies from-zone trust to-zone untrust policy block-ssh match destination-address any
set security policies from-zone trust to-zone untrust policy block-ssh match application app-ssh
set security policies from-zone trust to-zone untrust policy block-ssh then deny
set security ike policy IKE-POL pre-shared-key ascii-text SECRETPSKxyz`

func srx(t *testing.T, cfg string) Analysis {
	t.Helper()
	a, err := analyzeSrx(cfg)
	require.NoError(t, err)
	require.Len(t, a, 1)
	return a[0]
}

func TestSrxInterfacesZones(t *testing.T) {
	a := srx(t, srxCfg)
	require.Len(t, a.VLANs, 2)
	byName := map[string]VLAN{}
	for _, v := range a.VLANs {
		byName[v.Name] = v
	}
	require.Equal(t, "203.0.113.0/24", byName["ge-0/0/0.0"].Subnet)
	require.Equal(t, "10.0.20.0/24", byName["ge-0/0/1.20"].Subnet)
	require.Equal(t, 20, byName["ge-0/0/1.20"].ID)
	require.Equal(t, "trust", byName["ge-0/0/1.20"].Zone)

	zones := map[string][]string{}
	for _, z := range a.Zones {
		zones[z.Name] = z.Interfaces
	}
	require.Equal(t, []string{"ge-0/0/1.20"}, zones["trust"])
	require.Equal(t, []string{"ge-0/0/0.0"}, zones["untrust"])
}

func TestSrxAddressBookApplications(t *testing.T) {
	a := srx(t, srxCfg)
	addr := map[string]Address{}
	for _, x := range a.Addresses {
		addr[x.Name] = x
	}
	require.Equal(t, "subnet", addr["web-host"].Kind)
	require.Equal(t, "10.0.20.10/32", addr["web-host"].Value)
	require.Equal(t, "group", addr["servers"].Kind)
	require.Contains(t, addr["servers"].Value, "web-host")
	require.Contains(t, addr["servers"].Value, "dmz-net")

	svc := map[string]Service{}
	for _, s := range a.Services {
		svc[s.Name] = s
	}
	require.Equal(t, "TCP", svc["app-https"].Proto)
	require.Equal(t, "443", svc["app-https"].Ports)
	require.Equal(t, "group", svc["web-apps"].Proto)
}

func TestSrxPoliciesFlows(t *testing.T) {
	a := srx(t, srxCfg)
	require.Len(t, a.Policies, 2)
	byName := map[string]Policy{}
	for _, p := range a.Policies {
		byName[p.Name] = p
		require.NotZero(t, p.ID)
	}
	require.Equal(t, "accept", byName["allow-web"].Action)
	require.Equal(t, []string{"trust"}, byName["allow-web"].SrcIntf)
	require.Equal(t, []string{"untrust"}, byName["allow-web"].DstIntf)
	require.Equal(t, "deny", byName["block-ssh"].Action)

	var allow *Flow
	for i := range a.Flows {
		if a.Flows[i].Action == "allow" {
			allow = &a.Flows[i]
		}
	}
	require.NotNil(t, allow)
	require.Equal(t, internetNode, allow.Dst) // untrust is Internet-facing
	require.Contains(t, allow.Services, "app-https")
}

func TestSrxSecretNotSurfaced(t *testing.T) {
	a := srx(t, srxCfg)
	blob, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(blob), "SECRETPSKxyz")
}

func TestSrxRejectsCurlyBrace(t *testing.T) {
	_, err := analyzeSrx("security {\n    zones {\n        security-zone trust;\n    }\n}")
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "display set")
}
