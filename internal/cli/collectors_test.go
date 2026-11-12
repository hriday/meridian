package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/collector"
	"github.com/hriday/meridian/internal/netflow"
	"github.com/hriday/meridian/internal/snmp"
)

func TestFormatSyslog(t *testing.T) {
	out := formatSyslog(collector.SyslogMsg{
		T: time.Date(2026, 1, 1, 13, 14, 15, 0, time.UTC),
		Source: "10.0.0.9", SeverityName: "warning", Tag: "sshd", Message: "login failed",
	})
	for _, want := range []string{"13:14:15", "10.0.0.9", "warning", "sshd", "login failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatSyslog missing %q: %q", want, out)
		}
	}
}

func TestFormatTrap(t *testing.T) {
	out := formatTrap(snmp.TrapMsg{
		T: time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		Source: "192.0.2.1", Version: "v2c", TrapName: "linkDown",
		Varbinds: []snmp.TrapVar{{OID: "1.3.6.1.2.1.2.2.1.1", Value: "7"}},
	})
	for _, want := range []string{"09:00:00", "192.0.2.1", "v2c", "linkDown", "1.3.6.1.2.1.2.2.1.1=7"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatTrap missing %q: %q", want, out)
		}
	}
}

func TestFormatNetflow(t *testing.T) {
	src := netip.MustParseAddr("10.0.0.1")
	dst := netip.MustParseAddr("8.8.8.8")
	out := formatNetflow(netflow.FlowRecord{
		T: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Src: src, SrcPort: 51515, Dst: dst, DstPort: 53, Proto: 17, Packets: 3, Bytes: 210,
	})
	for _, want := range []string{"10.0.0.1:51515", "8.8.8.8:53", "UDP", "3 pkts", "210 bytes"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatNetflow missing %q: %q", want, out)
		}
	}
}

func TestProtoName(t *testing.T) {
	cases := map[uint8]string{1: "ICMP", 6: "TCP", 17: "UDP", 47: "GRE", 99: "proto99"}
	for p, want := range cases {
		if got := protoName(p); got != want {
			t.Errorf("protoName(%d) = %q, want %q", p, got, want)
		}
	}
}

func TestBindHintPrivileged(t *testing.T) {
	base := fmt.Errorf("permission denied")
	hint := bindHint(base, 514)
	if !strings.Contains(hint.Error(), "elevated privileges") {
		t.Errorf("privileged bind should hint at sudo/--port: %q", hint)
	}
	// A high port passes the error through unchanged.
	if got := bindHint(base, 2055); got.Error() != "permission denied" {
		t.Errorf("high port should not add a hint: %q", got)
	}
}

func TestCollectorCommandsRejectPositional(t *testing.T) {
	cases := map[string]func([]string) int{
		"syslog":  func(a []string) int { var o, e bytes.Buffer; return runSyslog(a, &o, &e) },
		"traps":   func(a []string) int { var o, e bytes.Buffer; return runTraps(a, &o, &e) },
		"netflow": func(a []string) int { var o, e bytes.Buffer; return runNetflow(a, &o, &e) },
	}
	for name, fn := range cases {
		if code := fn([]string{"stray-arg"}); code != 2 {
			t.Errorf("%s with a positional: got exit %d, want 2", name, code)
		}
	}
}

// End-to-end: bind a real UDP socket, send a syslog datagram, and confirm the
// streaming loop parses, formats, and prints it before the context is cancelled.
func TestStreamCollectorSyslogEndToEnd(t *testing.T) {
	old := collectorPoll
	collectorPoll = 10 * time.Millisecond
	defer func() { collectorPoll = old }()

	c := collector.New(100, collector.ParseSyslog)
	var out bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- streamCollector(ctx, &out, io.Discard, c, 0, false, formatSyslog) }()

	// Wait for the OS-chosen port to bind.
	var port int
	for i := 0; i < 200; i++ {
		if running, p := c.Status(); running {
			port = p
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if port == 0 {
		cancel()
		<-done
		t.Fatal("collector never bound")
	}

	conn, err := net.Dial("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		cancel()
		<-done
		t.Fatal(err)
	}
	_, _ = conn.Write([]byte("<34>meridian-cli test payload"))
	conn.Close()

	// Give the ring + poll loop time to drain.
	time.Sleep(120 * time.Millisecond)
	cancel()
	if code := <-done; code != 0 {
		t.Errorf("stream exit code %d, want 0", code)
	}
	if !strings.Contains(out.String(), "test payload") {
		t.Errorf("streamed output missing the message:\n%s", out.String())
	}
}
