package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dhcpscan"
)

func TestPrintDHCPSingle(t *testing.T) {
	var buf bytes.Buffer
	printDHCP(&buf, dhcpscan.Result{
		Interface: "en0",
		Servers: []dhcpscan.Server{{
			ServerID: "192.168.1.1", Offered: "192.168.1.50", Gateway: "192.168.1.1",
			DNS: []string{"192.168.1.1"}, LeaseSecs: 86400,
		}},
	})
	out := buf.String()
	for _, want := range []string{"1 DHCP server", "192.168.1.1", "192.168.1.50", "24h0m0s"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "ROGUE") {
		t.Error("single server must not be flagged rogue")
	}
}

func TestPrintDHCPRogue(t *testing.T) {
	var buf bytes.Buffer
	printDHCP(&buf, dhcpscan.Result{
		Interface: "en0", Rogue: true,
		Servers: []dhcpscan.Server{
			{ServerID: "192.168.1.1", Offered: "192.168.1.50"},
			{ServerID: "192.168.1.66", Offered: "10.0.0.5"},
		},
	})
	if !strings.Contains(buf.String(), "ROGUE") {
		t.Errorf("two servers should be flagged ROGUE: %s", buf.String())
	}
}

func TestPrintDHCPNone(t *testing.T) {
	var buf bytes.Buffer
	printDHCP(&buf, dhcpscan.Result{Interface: "en0"})
	if !strings.Contains(buf.String(), "No DHCP server answered") {
		t.Errorf("empty result should say none answered: %s", buf.String())
	}
}
