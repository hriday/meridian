package fwconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func findFlow(flows []Flow, src, dst, action string) *Flow {
	for i := range flows {
		if flows[i].Src == src && flows[i].Dst == dst && flows[i].Action == action {
			return &flows[i]
		}
	}
	return nil
}

func TestDeriveAllowAndDeny(t *testing.T) {
	a := analyze(t, `config firewall policy
    edit 1
        set srcintf "port1.30"
        set dstintf "port1.20"
        set service "HTTPS"
        set action accept
    next
    edit 2
        set srcintf "port1.30"
        set dstintf "port1.20"
        set service "SSH"
        set action deny
    next
end`)[0]
	require.Len(t, a.Flows, 2)
	require.NotNil(t, findFlow(a.Flows, "port1.30", "port1.20", "allow"))
	require.NotNil(t, findFlow(a.Flows, "port1.30", "port1.20", "deny"))
}

func TestDeriveZoneExpansion(t *testing.T) {
	a := analyze(t, `config system interface
    edit "port1.10"
        set vlanid 10
    next
    edit "port1.20"
        set vlanid 20
    next
end
config system zone
    edit "trust"
        set interface "port1.10" "port1.20"
    next
end
config firewall policy
    edit 1
        set srcintf "trust"
        set dstintf "port2.50"
        set action accept
    next
end`)[0]
	require.NotNil(t, findFlow(a.Flows, "port1.10", "port2.50", "allow"))
	require.NotNil(t, findFlow(a.Flows, "port1.20", "port2.50", "allow"))
}

func TestDeriveAnyAndWanToInternet(t *testing.T) {
	a := analyze(t, `config system interface
    edit "wan1"
        set role wan
    next
    edit "port1.30"
        set vlanid 30
    next
end
config firewall policy
    edit 1
        set srcintf "port1.30"
        set dstintf "wan1"
        set action accept
    next
    edit 2
        set srcintf "port1.30"
        set dstintf "any"
        set action accept
    next
end`)[0]
	// both the WAN interface and "any" collapse to the Internet node, and the
	// two policies aggregate into one flow.
	require.Len(t, a.Flows, 1)
	require.NotNil(t, findFlow(a.Flows, "port1.30", "Internet (any)", "allow"))
}

func TestDeriveFlowPolicyIDs(t *testing.T) {
	a := analyze(t, `config firewall policy
    edit 3
        set srcintf "port1.30"
        set dstintf "port1.20"
        set action accept
    next
    edit 5
        set srcintf "port1.30"
        set dstintf "port1.20"
        set action accept
    next
end`)[0]
	// two accept policies on the same pair aggregate to one flow carrying both IDs
	f := findFlow(a.Flows, "port1.30", "port1.20", "allow")
	require.NotNil(t, f)
	require.Equal(t, []int{3, 5}, f.PolicyIDs)
}

func TestDeriveSkipsDisabled(t *testing.T) {
	a := analyze(t, `config firewall policy
    edit 1
        set srcintf "a"
        set dstintf "b"
        set action accept
        set status disable
    next
end`)[0]
	require.Empty(t, a.Flows)
}

func TestDeriveNegatedPolicyNote(t *testing.T) {
	a := analyze(t, `config firewall policy
    edit 7
        set srcintf "a"
        set dstintf "b"
        set action accept
        set srcaddr-negate enable
    next
end`)[0]
	require.Empty(t, a.Flows)
	require.NotEmpty(t, a.Notes)
	require.Contains(t, a.Notes[0], "7")
}

func TestDeriveCounts(t *testing.T) {
	a := analyze(t, `config system interface
    edit "port1.10"
        set vlanid 10
        set ip 10.0.10.1 255.255.255.0
    next
    edit "port1.20"
        set vlanid 20
        set ip 10.0.20.1 255.255.255.0
    next
end
config system zone
    edit "trust"
        set interface "port1.10"
    next
end
config firewall address
    edit "web"
        set subnet 10.0.20.10 255.255.255.255
    next
    edit "all"
        set subnet 0.0.0.0 0.0.0.0
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
        set srcintf "port1.10"
        set dstintf "port1.20"
        set action accept
    next
end`)[0]
	require.Equal(t, 2, a.Counts.VLANs)
	require.Equal(t, 1, a.Counts.Policies)
	require.Equal(t, 2, a.Counts.Addresses)
	require.Equal(t, 1, a.Counts.Services)
	require.Equal(t, 1, a.Counts.Zones)
	// {10.0.10.0/24, 10.0.20.0/24, 10.0.20.10/32}; the "all" / 0.0.0.0/0 excluded
	require.Equal(t, 3, a.Counts.SubnetsIPs)
}
