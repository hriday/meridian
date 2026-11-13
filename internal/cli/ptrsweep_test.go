package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/ptrsweep"
)

func TestPrintPTRSweep(t *testing.T) {
	var buf bytes.Buffer
	printPTRSweep(&buf, []ptrsweep.Entry{
		{IP: "10.0.0.1", Names: []string{"gw.local"}},
		{IP: "10.0.0.5", Names: []string{"host5.local", "alias5.local"}},
	})
	out := buf.String()
	for _, want := range []string{"2 named host", "10.0.0.1", "gw.local", "10.0.0.5", "alias5.local"} {
		if !strings.Contains(out, want) {
			t.Errorf("printPTRSweep missing %q\n%s", want, out)
		}
	}
}

func TestRunPTRSweepMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runPTRSweep(nil, &o, &e); code != 2 {
		t.Errorf("ptrsweep with no args: got exit %d, want 2", code)
	}
}
