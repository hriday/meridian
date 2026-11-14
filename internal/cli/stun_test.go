package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/stun"
)

func TestPrintSTUN(t *testing.T) {
	var buf bytes.Buffer
	printSTUN(&buf, stun.Result{
		Server: "stun.l.google.com:19302", LocalIP: "192.168.1.82", PublicIP: "203.0.113.9",
		PublicPort: 54321, NAT: true, NATType: "Symmetric NAT — media needs a TURN relay", RTTMillis: 22,
	})
	out := buf.String()
	for _, want := range []string{"stun.l.google.com", "192.168.1.82", "203.0.113.9:54321", "Symmetric NAT"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}
