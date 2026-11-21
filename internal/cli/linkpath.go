package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/linkpath"
)

func runLinkPath(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("linkpath", flag.ContinueOnError)
	fs.SetOutput(stderr)
	wait := fs.Duration("wait", linkpath.DefaultLLDPWindow, "how long to listen for a switch's LLDP/CDP advertisement")
	asJSON := fs.Bool("json", false, "output JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), *wait+10*time.Second)
	defer cancel()
	rep, err := linkpath.Inspect(ctx, *wait)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printLinkPath(stdout, rep)
	return 0
}

func printLinkPath(w io.Writer, r linkpath.Report) {
	reportRow(w, "Egress", r.Iface)
	reportRow(w, "Gateway", r.Gateway)
	switch {
	case r.Wireless:
		reportRow(w, "Link", "Wi-Fi (wireless)")
	case r.LinkKnown:
		speed := ""
		if r.SpeedMbps > 0 {
			speed = fmt.Sprintf("%d Mb/s, ", r.SpeedMbps)
		}
		dup := r.Duplex
		if dup == "" {
			dup = "duplex unknown"
		} else {
			dup += " duplex"
		}
		reportRow(w, "Link", speed+dup)
	default:
		reportRow(w, "Link", "speed/duplex unknown on this system")
	}

	fmt.Fprintf(w, "\n%s\n\n", r.Verdict)
	for _, f := range r.Findings {
		mark := "·"
		switch f.Level {
		case "found":
			mark = "▶"
		case "good":
			mark = "✓"
		case "limit":
			mark = "?"
		}
		fmt.Fprintf(w, "  %s %s\n", mark, f.Text)
	}
}
