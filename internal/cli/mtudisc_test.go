package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/mtu"
)

func TestPrintPathMTUMeasured(t *testing.T) {
	var buf bytes.Buffer
	printPathMTU(&buf, mtu.Result{
		Target: "example.com", Addr: "203.0.113.5", DFEnforced: true,
		PathMTU: 1400, LocalMTU: 1500, Bottleneck: true,
	})
	out := buf.String()
	for _, want := range []string{"example.com", "1400", "1500", "Bottleneck"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintPathMTUNotEnforced(t *testing.T) {
	var buf bytes.Buffer
	printPathMTU(&buf, mtu.Result{
		Target: "example.com", Addr: "203.0.113.5", DFEnforced: false,
		LocalMTU: 1500, Note: "this system isn't enforcing Don't-Fragment",
	})
	out := buf.String()
	if !strings.Contains(out, "unmeasurable") || !strings.Contains(out, "Don't-Fragment") {
		t.Errorf("DF-not-enforced result should explain unmeasurability: %q", out)
	}
}

func TestRunPathMTUMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runPathMTU(nil, &o, &e); code != 2 {
		t.Errorf("pathmtu with no args: got exit %d, want 2", code)
	}
}
