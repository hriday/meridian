package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"flag"

	"github.com/hriday/meridian/internal/netquality"
	"github.com/hriday/meridian/internal/ping"
	"github.com/hriday/meridian/internal/scan"
)

func runQuality(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("quality", flag.ContinueOnError)
	fs.SetOutput(stderr)
	count := fs.Int("count", 25, "number of probes")
	interval := fs.Duration("interval", 200*time.Millisecond, "time between probes")
	timeout := fs.Duration("timeout", time.Second, "per-probe timeout")
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "quality needs exactly one host\n%s", usage)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Fprintf(stderr, "Measuring %s over %d probes…\n", pos[0], *count)

	// Warm the route/ARP first so a slow cold-start probe isn't counted as loss.
	_, _ = ping.Run(ctx, scan.NewICMPPinger(), pos[0], 2, 100*time.Millisecond, 2*time.Second, func(ping.Reply) {})

	s, err := ping.Run(ctx, scan.NewICMPPinger(), pos[0], *count, *interval, *timeout, func(ping.Reply) {})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !s.Available {
		fmt.Fprintln(stderr, "ICMP is unavailable (try elevated privileges)")
		return 1
	}
	if s.Received == 0 {
		fmt.Fprintf(stderr, "no replies from %s — host unreachable or not answering ICMP\n", pos[0])
		return 1
	}

	score := netquality.Rate(s.AvgMs, s.JitterMs, s.LossPct)
	if *asJSON {
		return writeJSON(stdout, stderr, struct {
			ping.Stats
			Score netquality.Score `json:"score"`
		}{s, score})
	}
	printQuality(stdout, s, score)
	return 0
}

func printQuality(w io.Writer, s ping.Stats, score netquality.Score) {
	reportRow(w, "Target", fmt.Sprintf("%s (%s)", s.Host, s.IP))
	reportRow(w, "Latency (avg)", fmt.Sprintf("%.1f ms  (min %.1f / max %.1f)", s.AvgMs, s.MinMs, s.MaxMs))
	reportRow(w, "Jitter", fmt.Sprintf("%.1f ms", s.JitterMs))
	reportRow(w, "Packet loss", fmt.Sprintf("%.0f%%  (%d/%d received)", s.LossPct, s.Received, s.Sent))
	reportRow(w, "MOS (estimated)", fmt.Sprintf("%.2f / 5  —  %s", score.MOS, score.Rating))
	fmt.Fprintf(w, "\n%s\n", score.Summary)
}
