package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/axfr"
	"github.com/hriday/meridian/internal/sysconf"
)

func runAXFR(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("axfr", flag.ContinueOnError)
	fs.SetOutput(stderr)
	resolver := fs.String("resolver", "", "resolver for the NS lookup (default: system)")
	timeout := fs.Duration("timeout", 20*time.Second, "overall timeout")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "axfr needs exactly one domain\n%s", usage)
		return 2
	}
	srv := *resolver
	if srv == "" {
		if f := sysconf.Resolvers(); len(f.Values) > 0 {
			srv = f.Values[0]
		} else {
			srv = "1.1.1.1"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	rep, err := axfr.Test(ctx, pos[0], srv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printAXFR(stdout, rep)
	if rep.Exposed {
		return 1 // make the exposure visible to scripts
	}
	return 0
}

func printAXFR(stdout io.Writer, r axfr.Report) {
	reportRow(stdout, "Domain", r.Domain)
	reportRow(stdout, "Summary", r.Summary)
	fmt.Fprintln(stdout)
	for _, ns := range r.Nameservers {
		fmt.Fprintf(stdout, "%-12s %-28s %s\n", string(ns.Status), ns.Nameserver, ns.Detail)
		for _, s := range ns.Sample {
			fmt.Fprintf(stdout, "             leaked: %s\n", s)
		}
	}
	if len(r.Remediation) > 0 {
		fmt.Fprintf(stdout, "\nHow to fix:\n")
		for _, s := range r.Remediation {
			fmt.Fprintf(stdout, "  • %s\n", s)
		}
	}
}
