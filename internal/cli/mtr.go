package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/hriday/meridian/internal/mtr"
	"github.com/hriday/meridian/internal/tracerun"
)

func runMTR(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mtr", flag.ContinueOnError)
	fs.SetOutput(stderr)
	interval := fs.Duration("interval", time.Second, "time between route sweeps")
	useV6 := fs.Bool("6", false, "trace over IPv6")
	asJSON := fs.Bool("json", false, "stream one JSON snapshot per round (ndjson)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "mtr needs exactly one host\n%s", usage)
		return 2
	}

	// Ctrl-C ends the monitor cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	enc := json.NewEncoder(stdout)
	lastLines := 0
	onUpdate := func(u mtr.Update) {
		if *asJSON {
			_ = enc.Encode(u)
			return
		}
		if lastLines > 0 {
			fmt.Fprintf(stdout, "\033[%dA", lastLines) // cursor up to redraw in place
		}
		lastLines = printMTRTable(stdout, u)
	}

	if err := mtr.Monitor(ctx, tracerun.Run, pos[0], *useV6, *interval, onUpdate); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

// printMTRTable renders the current snapshot as a table and returns the number
// of lines written (so the caller can redraw it in place next round).
func printMTRTable(w io.Writer, u mtr.Update) int {
	dest := u.Target
	if u.Addr != "" && u.Addr != u.Target {
		dest = fmt.Sprintf("%s (%s)", u.Target, u.Addr)
	}
	fmt.Fprintf(w, "mtr → %-40s  round %d\033[K\n", dest, u.Round)
	fmt.Fprintf(w, "%-34s %6s %4s %8s %8s %8s %8s\033[K\n", "Host", "Loss%", "Snt", "Last", "Avg", "Best", "Worst")
	for _, h := range u.Hops {
		fmt.Fprintf(w, "%2d. %-30s %5.1f%% %4d %8s %8s %8s %8s\033[K\n",
			h.TTL, hopLabel(h), h.LossPct, h.Sent,
			msOrDash(h.Recv, h.Last), msOrDash(h.Recv, h.Avg),
			msOrDash(h.Recv, h.Best), msOrDash(h.Recv, h.Worst))
	}
	return 2 + len(u.Hops)
}

func hopLabel(h mtr.HopStat) string {
	if h.Addr == "" {
		return "???"
	}
	label := h.Addr
	if h.Host != "" && h.Host != h.Addr {
		label = h.Host + " (" + h.Addr + ")"
	}
	if h.ASN != "" {
		label += " [" + h.ASN + "]"
	}
	if len(label) > 30 {
		label = label[:29] + "…"
	}
	return label
}

// msOrDash formats a duration in ms, or "--" when nothing has been received at
// this hop yet.
func msOrDash(recv int, d time.Duration) string {
	if recv == 0 {
		return "--"
	}
	return strconv.FormatFloat(float64(d)/float64(time.Millisecond), 'f', 1, 64) + "ms"
}
