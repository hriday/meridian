package fwconfig

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func analyze(t *testing.T, cfg string) []Analysis {
	t.Helper()
	root, err := parse(cfg)
	require.NoError(t, err)
	a, err := interpret(root)
	require.NoError(t, err)
	return a
}

func TestInterpretVLANsAndZones(t *testing.T) {
	cfg := `config system interface
    edit "port1.10"
        set vlanid 10
        set interface "port1"
        set ip 10.0.10.1 255.255.255.0
    next
    edit "port1.20"
        set vlanid 20
        set ip 10.0.20.1 255.255.255.0
    next
    edit "port1"
        set ip 192.168.1.1 255.255.255.0
    next
end
config system zone
    edit "trust"
        set interface "port1.10" "port1.20"
    next
end`
	a := analyze(t, cfg)
	require.Len(t, a, 1)
	require.Len(t, a[0].VLANs, 2) // port1 has no vlanid

	require.Equal(t, 10, a[0].VLANs[0].ID)
	require.Equal(t, "port1.10", a[0].VLANs[0].Name)
	require.Equal(t, "10.0.10.0/24", a[0].VLANs[0].Subnet)
	require.Equal(t, "trust", a[0].VLANs[0].Zone)

	require.Len(t, a[0].Zones, 1)
	require.Equal(t, []string{"port1.10", "port1.20"}, a[0].Zones[0].Interfaces)
}

func TestInterpretAddressesServicesPolicies(t *testing.T) {
	cfg := `config firewall address
    edit "web"
        set subnet 10.0.20.10 255.255.255.255
    next
    edit "rfc1918-range"
        set type iprange
        set start-ip 10.0.0.0
        set end-ip 10.0.0.255
    next
    edit "example.com"
        set type fqdn
        set fqdn "example.com"
    next
end
config firewall service custom
    edit "HTTPS"
        set protocol TCP
        set tcp-portrange 443
    next
end
config firewall policy
    edit 1
        set name "users-to-servers"
        set srcintf "port1.30"
        set dstintf "port1.20"
        set srcaddr "all"
        set dstaddr "web"
        set service "HTTPS"
        set action accept
    next
end`
	a := analyze(t, cfg)[0]
	require.Len(t, a.Addresses, 3)
	require.Equal(t, "web", a.Addresses[0].Name)
	require.Equal(t, "10.0.20.10/32", a.Addresses[0].Value)
	require.Equal(t, "iprange", a.Addresses[1].Kind)
	require.Contains(t, a.Addresses[1].Value, "10.0.0.0")
	require.Equal(t, "fqdn", a.Addresses[2].Kind)

	require.Len(t, a.Services, 1)
	require.Equal(t, "HTTPS", a.Services[0].Name)
	require.Equal(t, "TCP", a.Services[0].Proto)
	require.Equal(t, "443", a.Services[0].Ports)

	require.Len(t, a.Policies, 1)
	p := a.Policies[0]
	require.Equal(t, 1, p.ID)
	require.Equal(t, "users-to-servers", p.Name)
	require.Equal(t, []string{"port1.30"}, p.SrcIntf)
	require.Equal(t, []string{"port1.20"}, p.DstIntf)
	require.Equal(t, []string{"HTTPS"}, p.Services)
	require.Equal(t, "accept", p.Action)
}

func TestInterpretMultiVdom(t *testing.T) {
	cfg := `config vdom
edit root
    config system interface
        edit "port1.10"
            set vlanid 10
        next
    end
    config firewall policy
        edit 1
            set srcintf "port1.10"
            set dstintf "port1.20"
            set action accept
        next
    end
next
edit dmz
    config system interface
        edit "port2.50"
            set vlanid 50
        next
    end
next
end`
	a := analyze(t, cfg)
	require.Len(t, a, 2)
	require.Equal(t, "root", a[0].VDOM)
	require.Equal(t, "dmz", a[1].VDOM)
	require.Len(t, a[0].VLANs, 1)
	require.Len(t, a[0].Policies, 1)
	require.Len(t, a[1].VLANs, 1)
	require.Len(t, a[1].Policies, 0)
}

func TestInterpretSingleVdomName(t *testing.T) {
	a := analyze(t, `config system interface
    edit "port1.10"
        set vlanid 10
    next
end`)
	require.Len(t, a, 1)
	require.Equal(t, "", a[0].VDOM)
}

func TestInterpretSecretsExcluded(t *testing.T) {
	cfg := `config vpn ipsec phase1-interface
    edit "vpn1"
        set psksecret ENC abc123deadbeefSECRET
    next
end
config system interface
    edit "port1.10"
        set vlanid 10
    next
end`
	a := analyze(t, cfg)
	b, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(b), "abc123deadbeefSECRET")
	require.NotContains(t, string(b), "ENC ")
}

func TestInterpretEmptyConfig(t *testing.T) {
	a := analyze(t, "")
	require.Len(t, a, 1)
	require.Empty(t, a[0].VLANs)
	require.Empty(t, a[0].Policies)
	require.NotNil(t, a[0].VLANs) // non-nil slice so JSON emits []
}
