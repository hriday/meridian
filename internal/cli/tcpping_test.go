package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/tcpping"
)

func TestHostPortArgs(t *testing.T) {
	h, p, err := hostPortArgs([]string{"example.com", "443"})
	if err != nil || h != "example.com" || p != 443 {
		t.Errorf("two args: %q %d %v", h, p, err)
	}
	h, p, err = hostPortArgs([]string{"example.com:8443"})
	if err != nil || h != "example.com" || p != 8443 {
		t.Errorf("host:port: %q %d %v", h, p, err)
	}
	if _, _, err := hostPortArgs([]string{"example.com", "70000"}); err == nil {
		t.Error("expected error for out-of-range port")
	}
	if _, _, err := hostPortArgs(nil); err == nil {
		t.Error("expected error for no args")
	}
}

func TestPrintTCPPing(t *testing.T) {
	var buf bytes.Buffer
	printTCPPing(&buf, tcpping.Result{
		Host: "example.com", Port: 443, Address: "93.184.216.34:443",
		Sent: 3, Received: 3, LossPct: 0, MinMs: 10, AvgMs: 20, MaxMs: 30, JitterMs: 10,
		Verdict: "Port open and fast — avg 20.0 ms, no loss.",
		Probes:  []tcpping.Probe{{Seq: 1, RTTMs: 10, OK: true}},
	})
	out := buf.String()
	for _, want := range []string{"example.com:443", "93.184.216.34", "20.00", "Port open"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}
