package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/ping"
)

func TestPrintPingStats(t *testing.T) {
	s := ping.Stats{
		Host: "example.com", IP: "93.184.216.34",
		Sent: 5, Received: 4, LossPct: 20,
		MinMs: 10, AvgMs: 15, MaxMs: 25, StddevMs: 5.5, JitterMs: 7.25,
		Available: true,
	}
	var buf bytes.Buffer
	printPingStats(&buf, s)
	out := buf.String()
	for _, want := range []string{"93.184.216.34", "5", "4", "20", "15", "25", "7.25"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestPrintPingStatsUnavailable(t *testing.T) {
	var buf bytes.Buffer
	printPingStats(&buf, ping.Stats{Host: "h", IP: "1.2.3.4", Available: false})
	out := strings.ToLower(buf.String())
	if !strings.Contains(out, "icmp") || !strings.Contains(out, "available") {
		t.Errorf("unavailable output should explain ICMP isn't available, got:\n%s", buf.String())
	}
	if strings.Contains(out, "100% loss") {
		t.Errorf("unavailable must not read as 100%% loss:\n%s", buf.String())
	}
}

func TestRunPingMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runPing(nil, &out, &errb); code != 2 {
		t.Errorf("runPing with no args: got exit %d, want 2", code)
	}
}
