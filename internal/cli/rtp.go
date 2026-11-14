package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/hriday/meridian/internal/rtp"
)

func runRTP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rtp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	seconds := fs.Int("seconds", 10, "how long to capture")
	iface := fs.String("iface", "", "interface to capture on (default: the egress interface)")
	asJSON := fs.Bool("json", false, "output as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if !rtp.Supported() {
		fmt.Fprintln(stderr, "RTP capture isn't supported on this operating system yet")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Fprintf(stderr, "Capturing RTP for %ds (Ctrl-C to stop early)…\n", *seconds)

	streams, err := rtp.Capture(ctx, *iface, time.Duration(*seconds)*time.Second)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, streams)
	}
	printRTP(stdout, streams)
	return 0
}

func printRTP(w io.Writer, streams []rtp.Stream) {
	if len(streams) == 0 {
		fmt.Fprintln(w, "No RTP streams seen. You only capture RTP you're in the path of (your own calls, or a SPAN/mirror port).")
		return
	}
	fmt.Fprintf(w, "%d RTP stream(s):\n", len(streams))
	for i, s := range streams {
		if i > 0 {
			fmt.Fprintln(w)
		}
		reportRow(w, "SSRC", fmt.Sprintf("0x%08X", s.SSRC))
		reportRow(w, "Flow", fmt.Sprintf("%s → %s", s.Src, s.Dst))
		reportRow(w, "Codec", fmt.Sprintf("%s (PT %d)", s.Codec, s.PayloadType))
		reportRow(w, "Packets", fmt.Sprintf("%d over %dms", s.Packets, s.DurationMs))
		reportRow(w, "Loss", fmt.Sprintf("%d (%.1f%%)", s.Lost, s.LossPct))
		reportRow(w, "Jitter", fmt.Sprintf("%.2f ms", s.JitterMs))
	}
}
