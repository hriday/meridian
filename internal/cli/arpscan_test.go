package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/arpscan"
)

func TestPrintARP(t *testing.T) {
	var buf bytes.Buffer
	printARP(&buf, arpscan.Result{
		Interface: "en0", Subnet: "192.168.1.0/24",
		Hosts: []arpscan.Host{
			{IP: "192.168.1.1", MAC: "70:a7:41:a4:1a:6b", Vendor: "Ubee"},
			{IP: "192.168.1.144", MAC: "00:17:88:7e:18:22", Vendor: "Philips"},
		},
	})
	out := buf.String()
	for _, want := range []string{"2 host(s)", "192.168.1.0/24", "192.168.1.1", "70:a7:41:a4:1a:6b", "Philips"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintARPEmpty(t *testing.T) {
	var buf bytes.Buffer
	printARP(&buf, arpscan.Result{Interface: "en0", Subnet: "192.168.1.0/24"})
	if !strings.Contains(buf.String(), "0 host(s)") {
		t.Errorf("empty sweep should say 0 hosts: %s", buf.String())
	}
}
