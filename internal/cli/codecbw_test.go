package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/voipcalc"
)

func TestPrintCodecBW(t *testing.T) {
	var buf bytes.Buffer
	r, _ := voipcalc.Calc("G.711", 20, 23)
	printCodecBW(&buf, r)
	out := buf.String()
	for _, want := range []string{"G.711", "20ms", "80.0 kbps", "160.0 kbps", "23 calls"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunCodecBWUnknown(t *testing.T) {
	var o, e bytes.Buffer
	if code := runCodecBW([]string{"MP3"}, &o, &e); code != 1 {
		t.Errorf("unknown codec should exit 1, got %d", code)
	}
}

func TestRunCodecBWNoArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runCodecBW(nil, &o, &e); code != 2 {
		t.Errorf("no codec should exit 2, got %d", code)
	}
}
