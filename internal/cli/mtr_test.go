package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/mtr"
)

func TestPrintMTRTable(t *testing.T) {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	var buf bytes.Buffer
	n := printMTRTable(&buf, mtr.Update{
		Round: 3, Target: "example.com", Addr: "203.0.113.5", Reached: true,
		Hops: []mtr.HopStat{
			{TTL: 1, Addr: "10.0.0.1", Host: "gw", Sent: 3, Recv: 3, LossPct: 0, Last: ms(2), Avg: ms(2), Best: ms(2), Worst: ms(3)},
			{TTL: 2, Addr: "203.0.113.5", ASN: "AS64500", Sent: 3, Recv: 2, LossPct: 33.3, Last: ms(10), Avg: ms(11), Best: ms(9), Worst: ms(14)},
		},
	})
	out := buf.String()
	for _, want := range []string{"example.com", "203.0.113.5", "Loss%", "gw", "AS64500", "33.3%", "2.0ms", "round 3"} {
		if !strings.Contains(out, want) {
			t.Errorf("printMTRTable missing %q\n%s", want, out)
		}
	}
	if n != 4 { // header + title + 2 hops
		t.Errorf("printMTRTable reported %d lines, want 4", n)
	}
}

func TestPrintMTRTablePendingHop(t *testing.T) {
	var buf bytes.Buffer
	printMTRTable(&buf, mtr.Update{
		Round: 1, Target: "example.com",
		Hops: []mtr.HopStat{{TTL: 1, Sent: 1, Recv: 0, LossPct: 100}},
	})
	out := buf.String()
	if !strings.Contains(out, "???") {
		t.Errorf("hop with no reply should show ???: %q", out)
	}
	if !strings.Contains(out, "--") {
		t.Errorf("hop with no RTT should show -- for timings: %q", out)
	}
}

func TestRunMTRMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runMTR(nil, &o, &e); code != 2 {
		t.Errorf("mtr with no args: got exit %d, want 2", code)
	}
}
