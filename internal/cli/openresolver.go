package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/openresolver"
)

func runOpenResolver(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("openresolver", flag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 6*time.Second, "query timeout")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "openresolver needs exactly one resolver (ip or host)\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	rep, err := openresolver.Test(ctx, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	reportRow(stdout, "Resolver", rep.Resolver)
	reportRow(stdout, "Probe", rep.Probe+" (A)")
	reportRow(stdout, "Rcode", rep.Rcode)
	reportRow(stdout, "Summary", rep.Summary)
	if rep.Open {
		return 1 // surface the finding to scripts
	}
	return 0
}
