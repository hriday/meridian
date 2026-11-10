package fwconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNestedConfigEdit(t *testing.T) {
	cfg := `config system interface
    edit "port1.10"
        set vlanid 10
        set ip 10.0.10.1 255.255.255.0
    next
    edit "port1.20"
        set vlanid 20
    next
end`
	root, err := parse(cfg)
	require.NoError(t, err)

	ifs := root.configs("system interface")
	require.Len(t, ifs, 1)
	edits := ifs[0].edits()
	require.Len(t, edits, 2)

	require.Equal(t, "port1.10", edits[0].Name)
	v, ok := edits[0].get("vlanid")
	require.True(t, ok)
	require.Equal(t, "10", v)
	ip, ok := edits[0].get("ip")
	require.True(t, ok)
	require.Equal(t, "10.0.10.1 255.255.255.0", ip)

	require.Equal(t, "port1.20", edits[1].Name)
}

func TestParseQuotedValues(t *testing.T) {
	cfg := `config firewall address
    edit "web server farm"
        set comment "dmz \"public\" hosts"
        set subnet 203.0.113.0 255.255.255.240
    next
end`
	root, err := parse(cfg)
	require.NoError(t, err)
	e := root.configs("firewall address")[0].edits()
	require.Len(t, e, 1)
	require.Equal(t, "web server farm", e[0].Name)
	c, _ := e[0].get("comment")
	require.Equal(t, `dmz "public" hosts`, c)
}

func TestParseMultiVdom(t *testing.T) {
	cfg := `config global
    config system global
        set hostname "fw1"
    end
end
config vdom
edit root
    config system interface
        edit "port1.10"
            set vlanid 10
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
	root, err := parse(cfg)
	require.NoError(t, err)

	vdoms := root.configs("vdom")
	require.Len(t, vdoms, 1)
	ve := vdoms[0].edits()
	require.Len(t, ve, 2)
	require.Equal(t, "root", ve[0].Name)
	require.Equal(t, "dmz", ve[1].Name)
	// each vdom holds its own interface section
	require.Len(t, ve[0].configs("system interface"), 1)
	require.Len(t, ve[1].configs("system interface"), 1)
}

func TestParseIgnoresCommentsAndHeader(t *testing.T) {
	cfg := `#config-version=FGT60F-7.0.5
# a comment
config system zone
    edit "trust"
        set interface "port1.10" "port1.20"
    next
end`
	root, err := parse(cfg)
	require.NoError(t, err)
	z := root.configs("system zone")[0].edits()[0]
	require.Equal(t, "trust", z.Name)
	ifaces := z.getAll("interface")
	require.Equal(t, []string{"port1.10", "port1.20"}, ifaces)
}
