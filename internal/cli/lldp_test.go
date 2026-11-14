package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/lldp"
)

func TestPrintLLDPNeighbors(t *testing.T) {
	var buf bytes.Buffer
	printLLDP(&buf, []lldp.Neighbor{{
		Protocol: "LLDP", SystemName: "core-sw-1", PortID: "Gi0/1", PortDesc: "GigabitEthernet0/1",
		VLAN: 100, MgmtIP: "192.168.1.2", ChassisID: "00:11:22:33:44:55",
		Capabilities: []string{"Bridge", "Router"},
	}})
	out := buf.String()
	for _, want := range []string{"1 neighbor", "core-sw-1", "Gi0/1", "GigabitEthernet0/1", "100", "192.168.1.2", "Bridge"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintLLDPNone(t *testing.T) {
	var buf bytes.Buffer
	printLLDP(&buf, nil)
	if !strings.Contains(buf.String(), "No LLDP/CDP neighbors heard") {
		t.Errorf("empty result should explain: %s", buf.String())
	}
}
