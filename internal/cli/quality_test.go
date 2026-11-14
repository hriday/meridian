package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/netquality"
	"github.com/hriday/meridian/internal/ping"
)

func TestPrintQuality(t *testing.T) {
	var buf bytes.Buffer
	s := ping.Stats{Host: "1.1.1.1", IP: "1.1.1.1", Sent: 25, Received: 25, AvgMs: 18.2, MinMs: 15, MaxMs: 22, JitterMs: 1.4, LossPct: 0}
	printQuality(&buf, s, netquality.Rate(s.AvgMs, s.JitterMs, s.LossPct))
	out := buf.String()
	for _, want := range []string{"1.1.1.1", "Latency", "Jitter", "1.4 ms", "Packet loss", "MOS", "Excellent"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunQualityMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runQuality(nil, &o, &e); code != 2 {
		t.Errorf("no args: got exit %d, want 2", code)
	}
}
