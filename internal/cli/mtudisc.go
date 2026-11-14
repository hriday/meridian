package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/mtu"
)

func runPathMTU(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pathmtu", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output the result as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "pathmtu needs exactly one host\n%s", usage)
		return 2
	}
	if !mtu.Supported() {
		fmt.Fprintln(stderr, "path MTU discovery isn't supported on this operating system yet")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := mtu.DiscoverHost(ctx, mtu.Options{Target: pos[0]})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printPathMTU(stdout, res)
	return 0
}

func printPathMTU(w io.Writer, r mtu.Result) {
	reportRow(w, "Target", fmt.Sprintf("%s (%s)", r.Target, r.Addr))
	if !r.DFEnforced {
		reportRow(w, "Path MTU", "unmeasurable here")
		fmt.Fprintf(w, "\n%s\n", r.Note)
		return
	}
	reportRow(w, "Path MTU", fmt.Sprintf("%d bytes", r.PathMTU))
	if r.LocalMTU > 0 {
		reportRow(w, "Local MTU", fmt.Sprintf("%d bytes", r.LocalMTU))
	}
	if r.NextHopMTU > 0 {
		reportRow(w, "Next-hop MTU", fmt.Sprintf("%d bytes (reported by a router)", r.NextHopMTU))
	}
	if r.Bottleneck {
		fmt.Fprintf(w, "\nBottleneck: the path only carries %d bytes, below your %d-byte link — "+
			"full-size packets with Don't-Fragment set will be dropped.\n", r.PathMTU, r.LocalMTU)
	} else {
		fmt.Fprintf(w, "\nNo bottleneck: the path carries your full %d-byte link MTU.\n", r.PathMTU)
	}
}
