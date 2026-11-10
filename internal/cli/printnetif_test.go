package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/netif"
)

func TestPrintInterfaces(t *testing.T) {
	var buf bytes.Buffer
	printInterfaces(&buf, []netif.Iface{{
		Index: 2, Name: "en0", MAC: "aa:bb:cc:dd:ee:ff", MTU: 1500,
		Up: true, Flags: []string{"up", "broadcast"}, IPv4: []string{"192.168.1.20/24"},
	}})
	out := buf.String()
	for _, want := range []string{"en0", "aa:bb:cc:dd:ee:ff", "1500", "up", "192.168.1.20/24"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRunInterfacesRejectsArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runInterfaces([]string{"en0"}, &out, &errb); code != 2 {
		t.Errorf("runInterfaces with a stray arg: got exit %d, want 2", code)
	}
}

func TestRunInterfacesJSON(t *testing.T) {
	var out, errb bytes.Buffer
	code := runInterfaces([]string{"--json"}, &out, &errb)
	if code != 0 {
		t.Fatalf("runInterfaces --json: exit %d, stderr=%s", code, errb.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(out.String()), "[") {
		t.Errorf("--json output should be a JSON array, got: %s", out.String())
	}
}
