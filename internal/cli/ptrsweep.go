package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/ptrsweep"
)

func runPTRSweep(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ptrsweep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	conc := fs.Int("concurrency", 64, "parallel lookups")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "ptrsweep needs exactly one CIDR\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resolver := func(ctx context.Context, ip string) ([]string, error) {
		return net.DefaultResolver.LookupAddr(ctx, ip)
	}
	entries, err := ptrsweep.Sweep(ctx, resolver, pos[0], *conc, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, entries)
	}
	printPTRSweep(stdout, entries)
	return 0
}

func printPTRSweep(stdout io.Writer, entries []ptrsweep.Entry) {
	fmt.Fprintf(stdout, "%d named host(s)\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(stdout, "%-16s %s\n", e.IP, strings.Join(e.Names, ", "))
	}
}
