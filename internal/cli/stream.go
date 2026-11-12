package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/hriday/meridian/internal/ping"
	"github.com/hriday/meridian/internal/scan"
	"github.com/hriday/meridian/internal/speedtest"
)

// ---- pingmon ---------------------------------------------------------------

func runPingMon(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pingmon", flag.ContinueOnError)
	fs.SetOutput(stderr)
	interval := fs.Duration("interval", time.Second, "time between pings")
	timeout := fs.Duration("timeout", time.Second, "per-ping timeout")
	asJSON := fs.Bool("json", false, "stream one JSON object per reply (ndjson)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "pingmon needs exactly one host\n%s", usage)
		return 2
	}

	// Ctrl-C ends the monitor cleanly and prints the summary.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var replies []ping.Reply
	enc := json.NewEncoder(stdout)
	onReply := func(r ping.Reply) {
		replies = append(replies, r)
		if *asJSON {
			_ = enc.Encode(r)
			return
		}
		fmt.Fprintln(stdout, pingReplyLine(pos[0], r))
	}

	avail, err := ping.Monitor(ctx, scan.NewICMPPinger(), pos[0], *interval, *timeout, onReply)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !avail {
		fmt.Fprintln(stderr, "ICMP is unavailable (try running with elevated privileges)")
		return 1
	}
	if !*asJSON {
		printPingMonSummary(stdout, pos[0], replies)
	}
	return 0
}

func pingReplyLine(host string, r ping.Reply) string {
	if !r.OK {
		return fmt.Sprintf("%-20s seq=%-4d timeout", host, r.Seq)
	}
	return fmt.Sprintf("%-20s seq=%-4d %.1f ms", host, r.Seq, r.RTTMillis)
}

func printPingMonSummary(stdout io.Writer, host string, replies []ping.Reply) {
	sent, recv, min, avg, max := summarizePing(replies)
	loss := 100.0
	if sent > 0 {
		loss = float64(sent-recv) / float64(sent) * 100
	}
	fmt.Fprintf(stdout, "\n--- %s ping statistics ---\n", host)
	fmt.Fprintf(stdout, "%d sent, %d received, %.0f%% loss\n", sent, recv, loss)
	if recv > 0 {
		fmt.Fprintf(stdout, "rtt min/avg/max = %.1f/%.1f/%.1f ms\n", min, avg, max)
	}
}

// summarizePing returns sent, received, and min/avg/max RTT (ms) over replies.
func summarizePing(replies []ping.Reply) (sent, recv int, min, avg, max float64) {
	min = math.Inf(1)
	var sum float64
	for _, r := range replies {
		sent++
		if !r.OK {
			continue
		}
		recv++
		sum += r.RTTMillis
		if r.RTTMillis < min {
			min = r.RTTMillis
		}
		if r.RTTMillis > max {
			max = r.RTTMillis
		}
	}
	if recv > 0 {
		avg = sum / float64(recv)
	} else {
		min = 0
	}
	return sent, recv, min, avg, max
}

// ---- speedtest -------------------------------------------------------------

func runSpeedTest(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("speedtest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 5201, "iperf3 server port")
	dir := fs.String("dir", "download", "download | upload | bidir")
	parallel := fs.Int("parallel", 1, "parallel streams")
	dur := fs.Duration("duration", 10*time.Second, "test duration")
	udp := fs.Bool("udp", false, "use UDP instead of TCP")
	asJSON := fs.Bool("json", false, "output the final result as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "speedtest needs exactly one server host (an iperf3 server)\n%s", usage)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	onSample := func(s speedtest.Sample) {
		if *asJSON {
			return // final Result is the JSON payload
		}
		if s.Phase == "connecting" {
			return
		}
		fmt.Fprintf(stdout, "\r%5.1fs  ↓ %7.1f Mbps  ↑ %7.1f Mbps", s.Elapsed, s.DownMbps, s.UpMbps)
	}

	res, err := speedtest.Run(ctx, speedtest.Options{
		Host: pos[0], Port: *port, Direction: *dir, Parallel: *parallel, Duration: *dur, UDP: *udp,
	}, onSample)
	if !*asJSON {
		fmt.Fprintln(stdout) // end the \r progress line
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printSpeedResult(stdout, res)
	return 0
}

func printSpeedResult(stdout io.Writer, r speedtest.Result) {
	reportRow(stdout, "Peer", r.Peer)
	reportRow(stdout, "Protocol", r.Protocol)
	if r.DownMbps > 0 {
		reportRow(stdout, "Download", fmt.Sprintf("%.1f Mbps", r.DownMbps))
	}
	if r.UpMbps > 0 {
		reportRow(stdout, "Upload", fmt.Sprintf("%.1f Mbps", r.UpMbps))
	}
	if r.RTTMeanMs > 0 {
		reportRow(stdout, "RTT (mean)", fmt.Sprintf("%.1f ms", r.RTTMeanMs))
	}
	if r.JitterMs > 0 {
		reportRow(stdout, "Jitter", fmt.Sprintf("%.2f ms", r.JitterMs))
	}
	if r.Retransmits > 0 {
		reportRow(stdout, "Retransmits", fmt.Sprintf("%d", r.Retransmits))
	}
	if r.Packets > 0 {
		reportRow(stdout, "Packets", fmt.Sprintf("%d (%d lost)", r.Packets, r.Lost))
	}
}
