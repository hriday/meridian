package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/netdiag"
)

// netdiagRun is a seam so tests can fake the engine without real probes.
var netdiagRun = netdiag.Run

// runNetDiag diagnoses why local-network resources are unreachable, using the
// same engine as the GUI.
//
// Exit codes: 0 = informational/healthy verdict (all_clear, host_down: the
// local network is fine); 1 = a problem verdict or engine error; 2 = usage.
func runNetDiag(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("netdiag", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) > 1 {
		fmt.Fprintf(stderr, "netdiag takes at most one host\n%s", usage)
		return 2
	}
	opts := netdiag.Options{}
	if len(pos) == 1 {
		opts.Target = pos[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	r, err := netdiagRun(ctx, opts, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	code := netdiagExit(r.Verdict)
	if *asJSON {
		if c := writeJSON(stdout, stderr, r); c != 0 {
			return c
		}
		return code
	}
	printNetDiag(stdout, r)
	return code
}

func netdiagExit(v netdiag.Verdict) int {
	switch v {
	case netdiag.VerdictAllClear, netdiag.VerdictHostDown:
		return 0
	}
	return 1
}

func printNetDiag(w io.Writer, r netdiag.Report) {
	reportRow(w, "Verdict", string(r.Verdict))
	reportRow(w, "Summary", r.Statement)
	if len(r.Steps) > 0 {
		fmt.Fprintln(w, "\nSteps:")
		for _, s := range r.Steps {
			line := fmt.Sprintf("  %-6s %s", s.Status, s.Name)
			if s.Detail != "" {
				line += " - " + s.Detail
			}
			fmt.Fprintln(w, line)
		}
	}
	if len(r.Causes) > 0 {
		fmt.Fprintln(w, "\nLikely causes:")
		for _, c := range r.Causes {
			fmt.Fprintf(w, "  - %s\n", c)
		}
	}
	if len(r.Fixes) > 0 {
		fmt.Fprintln(w, "\nHow to fix:")
		for _, f := range r.Fixes {
			fmt.Fprintf(w, "  - %s\n", f)
		}
	}
}
