package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/dnsprop"
	"github.com/hriday/meridian/internal/dnsx"
)

func runDNSProp(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dnspropagation", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rtype := fs.String("type", "A", "record type (A, AAAA, MX, NS, TXT, CNAME, SOA, …)")
	asJSON := fs.Bool("json", false, "output the full report as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "dnspropagation needs exactly one name\n%s", usage)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rep := dnsprop.Check(ctx, dnsx.Query, pos[0], *rtype, dnsprop.Public())

	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printDNSProp(stdout, rep)
	return 0
}

func printDNSProp(w io.Writer, rep dnsprop.Report) {
	verdict := "DIVERGED"
	if rep.Consensus {
		verdict = "CONSENSUS"
	}
	fmt.Fprintf(w, "%s %s — %d/%d resolvers agree [%s]\n\n",
		rep.Name, rep.Type, rep.Propagated, rep.Total, verdict)

	for i, g := range rep.Groups {
		val := strings.Join(g.Values, ", ")
		if val == "" {
			val = g.Rcode
		}
		tag := "majority"
		if i > 0 {
			tag = "differs"
		}
		fmt.Fprintf(w, "  [%s] %s\n", tag, val)
		fmt.Fprintf(w, "        %d resolver(s): %s\n", g.Count, strings.Join(g.Resolvers, ", "))
	}

	// Call out any resolver that failed to answer at all.
	var failed []string
	for _, a := range rep.Answers {
		if a.Err != "" {
			failed = append(failed, a.Resolver.Name)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintf(w, "\n  no answer from: %s\n", strings.Join(failed, ", "))
	}
}
