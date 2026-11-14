package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/hriday/meridian/internal/tcpping"
)

func runTCPPing(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tcpping", flag.ContinueOnError)
	fs.SetOutput(stderr)
	count := fs.Int("count", 5, "number of connect attempts")
	interval := fs.Duration("interval", 500*time.Millisecond, "delay between attempts")
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}

	host, port, err := hostPortArgs(pos)
	if err != nil {
		fmt.Fprintln(stderr, err)
		fmt.Fprintln(stderr, "usage: tcpping <host> <port>   (or host:port) [--count 5] [--interval 500ms] [--json]")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*count)*4*time.Second+10*time.Second)
	defer cancel()
	r, err := tcpping.Ping(ctx, tcpConnectCLI, host, port, *count, *interval)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	printTCPPing(stdout, r)
	if r.Received == 0 {
		return 1
	}
	return 0
}

// hostPortArgs accepts either ["host", "port"] or ["host:port"].
func hostPortArgs(pos []string) (string, int, error) {
	switch len(pos) {
	case 1:
		h, p, err := net.SplitHostPort(pos[0])
		if err != nil {
			return "", 0, fmt.Errorf("give a host and port: %q", pos[0])
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return "", 0, fmt.Errorf("bad port %q", p)
		}
		return h, n, nil
	case 2:
		n, err := strconv.Atoi(pos[1])
		if err != nil || n < 1 || n > 65535 {
			return "", 0, fmt.Errorf("bad port %q", pos[1])
		}
		return pos[0], n, nil
	default:
		return "", 0, fmt.Errorf("need a host and a port")
	}
}

func tcpConnectCLI(ctx context.Context, addr string) (time.Duration, string, error) {
	d := net.Dialer{Timeout: 3 * time.Second}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", addr)
	dur := time.Since(start)
	if err != nil {
		return dur, "", err
	}
	remote := c.RemoteAddr().String()
	_ = c.Close()
	return dur, remote, nil
}

func printTCPPing(w io.Writer, r tcpping.Result) {
	target := fmt.Sprintf("%s:%d", r.Host, r.Port)
	if r.Address != "" {
		target += "  (" + r.Address + ")"
	}
	reportRow(w, "Target", target)
	for _, p := range r.Probes {
		if p.OK {
			reportRow(w, fmt.Sprintf("seq %d", p.Seq), fmt.Sprintf("%.2f ms", p.RTTMs))
		} else {
			reportRow(w, fmt.Sprintf("seq %d", p.Seq), "failed — "+p.Err)
		}
	}
	reportRow(w, "Sent/received", fmt.Sprintf("%d/%d  (%.0f%% loss)", r.Sent, r.Received, r.LossPct))
	if r.Received > 0 {
		reportRow(w, "min/avg/max", fmt.Sprintf("%.2f / %.2f / %.2f ms", r.MinMs, r.AvgMs, r.MaxMs))
		reportRow(w, "Jitter", fmt.Sprintf("%.2f ms", r.JitterMs))
	}
	reportRow(w, "Verdict", r.Verdict)
}
