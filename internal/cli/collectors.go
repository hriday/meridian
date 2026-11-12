package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/collector"
	"github.com/hriday/meridian/internal/netflow"
	"github.com/hriday/meridian/internal/snmp"
)

// collectorPoll is how often the streaming loop drains the ring; a seam so tests
// don't wait 400ms per record.
var collectorPoll = 400 * time.Millisecond

// streamCollector binds c on port, prints each new record (ndjson when asJSON,
// else via format), and runs until ctx is cancelled (Ctrl-C). It owns Start/Stop.
func streamCollector[T any](ctx context.Context, stdout, stderr io.Writer, c *collector.Collector[T], port int, asJSON bool, format func(T) string) int {
	if err := c.Start(port); err != nil {
		fmt.Fprintln(stderr, bindHint(err, port))
		return 1
	}
	defer c.Stop()
	if _, bound := c.Status(); !asJSON {
		fmt.Fprintf(stderr, "listening on udp/%d — Ctrl-C to stop\n", bound)
	}

	enc := json.NewEncoder(stdout)
	ticker := time.NewTicker(collectorPoll)
	defer ticker.Stop()
	last := 0
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
			p := c.PollSince(last)
			for _, it := range p.Items {
				last = it.Seq
				if asJSON {
					_ = enc.Encode(it.Val)
				} else {
					fmt.Fprintln(stdout, format(it.Val))
				}
			}
		}
	}
}

// bindHint annotates a privileged-port bind failure with how to work around it.
func bindHint(err error, port int) error {
	if port > 0 && port < 1024 {
		return fmt.Errorf("%w — binding udp/%d needs elevated privileges; run with sudo, or pass --port <high-port> and point the sender there", err, port)
	}
	return err
}

func signalCtx() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

// ---- syslog ----------------------------------------------------------------

func runSyslog(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("syslog", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 514, "UDP port to bind")
	asJSON := fs.Bool("json", false, "stream one JSON object per message (ndjson)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 0 {
		fmt.Fprintf(stderr, "syslog takes no arguments (use --port)\n%s", usage)
		return 2
	}
	ctx, stop := signalCtx()
	defer stop()
	c := collector.New(2000, collector.ParseSyslog)
	return streamCollector(ctx, stdout, stderr, c, *port, *asJSON, formatSyslog)
}

func formatSyslog(m collector.SyslogMsg) string {
	tag := m.Tag
	if tag == "" {
		tag = "-"
	}
	return fmt.Sprintf("%s %-15s %-9s %-14s %s",
		m.T.Format("15:04:05"), m.Source, m.SeverityName, tag, m.Message)
}

// ---- traps -----------------------------------------------------------------

func runTraps(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("traps", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 162, "UDP port to bind")
	asJSON := fs.Bool("json", false, "stream one JSON object per trap (ndjson)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 0 {
		fmt.Fprintf(stderr, "traps takes no arguments (use --port)\n%s", usage)
		return 2
	}
	ctx, stop := signalCtx()
	defer stop()
	c := collector.New(2000, snmp.ParseTrap)
	return streamCollector(ctx, stdout, stderr, c, *port, *asJSON, formatTrap)
}

func formatTrap(m snmp.TrapMsg) string {
	name := m.TrapName
	if name == "" {
		name = m.TrapOID
	}
	var vb []string
	for i, v := range m.Varbinds {
		if i >= 4 {
			vb = append(vb, "…")
			break
		}
		vb = append(vb, v.OID+"="+v.Value)
	}
	return fmt.Sprintf("%s %-15s %-8s %s  [%s]",
		m.T.Format("15:04:05"), m.Source, m.Version, name, strings.Join(vb, ", "))
}

// ---- netflow ---------------------------------------------------------------

func runNetflow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("netflow", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 2055, "UDP port to bind")
	asJSON := fs.Bool("json", false, "stream one JSON object per flow (ndjson)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 0 {
		fmt.Fprintf(stderr, "netflow takes no arguments (use --port)\n%s", usage)
		return 2
	}
	ctx, stop := signalCtx()
	defer stop()
	parse := func(b []byte, src netip.Addr) []netflow.FlowRecord {
		return netflow.ParseV5(b, src, time.Now())
	}
	c := collector.NewMulti(5000, parse)
	return streamCollector(ctx, stdout, stderr, c, *port, *asJSON, formatNetflow)
}

func formatNetflow(f netflow.FlowRecord) string {
	return fmt.Sprintf("%s  %-21s → %-21s %-6s %7d pkts %12d bytes",
		f.T.Format("15:04:05"),
		fmt.Sprintf("%s:%d", f.Src, f.SrcPort),
		fmt.Sprintf("%s:%d", f.Dst, f.DstPort),
		protoName(f.Proto), f.Packets, f.Bytes)
}

func protoName(p uint8) string {
	switch p {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	case 47:
		return "GRE"
	case 50:
		return "ESP"
	default:
		return fmt.Sprintf("proto%d", p)
	}
}
