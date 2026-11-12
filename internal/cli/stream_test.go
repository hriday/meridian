package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/ping"
	"github.com/hriday/meridian/internal/speedtest"
)

func TestPingReplyLine(t *testing.T) {
	ok := pingReplyLine("1.1.1.1", ping.Reply{Seq: 3, RTTMillis: 12.4, OK: true})
	if !strings.Contains(ok, "seq=3") || !strings.Contains(ok, "12.4 ms") {
		t.Errorf("ok line wrong: %q", ok)
	}
	lost := pingReplyLine("1.1.1.1", ping.Reply{Seq: 4, OK: false})
	if !strings.Contains(lost, "timeout") {
		t.Errorf("timeout line wrong: %q", lost)
	}
}

func TestSummarizePing(t *testing.T) {
	replies := []ping.Reply{
		{Seq: 0, RTTMillis: 10, OK: true},
		{Seq: 1, OK: false},
		{Seq: 2, RTTMillis: 20, OK: true},
		{Seq: 3, RTTMillis: 30, OK: true},
	}
	sent, recv, min, avg, max := summarizePing(replies)
	if sent != 4 || recv != 3 {
		t.Errorf("sent/recv = %d/%d, want 4/3", sent, recv)
	}
	if min != 10 || max != 30 || avg != 20 {
		t.Errorf("min/avg/max = %.0f/%.0f/%.0f, want 10/20/30", min, avg, max)
	}
}

func TestSummarizePingAllLost(t *testing.T) {
	_, recv, min, avg, _ := summarizePing([]ping.Reply{{OK: false}, {OK: false}})
	if recv != 0 || min != 0 || avg != 0 {
		t.Errorf("all-lost should zero the rtt fields, got recv=%d min=%.0f avg=%.0f", recv, min, avg)
	}
}

func TestPrintPingMonSummary(t *testing.T) {
	var buf bytes.Buffer
	printPingMonSummary(&buf, "host.example", []ping.Reply{
		{RTTMillis: 10, OK: true}, {OK: false}, {RTTMillis: 30, OK: true},
	})
	out := buf.String()
	for _, want := range []string{"host.example", "3 sent", "2 received", "33% loss", "10.0/20.0/30.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n%s", want, out)
		}
	}
}

func TestPrintSpeedResult(t *testing.T) {
	var buf bytes.Buffer
	printSpeedResult(&buf, speedtest.Result{
		Peer: "10.0.0.5", Protocol: "TCP", DownMbps: 942.3, RTTMeanMs: 0.4, Retransmits: 12,
	})
	out := buf.String()
	for _, want := range []string{"10.0.0.5", "TCP", "942.3 Mbps", "0.4 ms", "12"} {
		if !strings.Contains(out, want) {
			t.Errorf("printSpeedResult missing %q\n%s", want, out)
		}
	}
}

func TestStreamCommandsMissingArg(t *testing.T) {
	cases := map[string]func([]string) int{
		"pingmon":   func(a []string) int { var o, e bytes.Buffer; return runPingMon(a, &o, &e) },
		"speedtest": func(a []string) int { var o, e bytes.Buffer; return runSpeedTest(a, &o, &e) },
	}
	for name, fn := range cases {
		if code := fn(nil); code != 2 {
			t.Errorf("%s with no args: got exit %d, want 2", name, code)
		}
	}
}
