package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/ikeprobe"
)

func runIke(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ike", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "ike needs a host (VPN gateway)\n%s", usage)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := ikeprobe.Probe(ctx, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	reportRow(stdout, "Target", res.Target)
	reportRow(stdout, "Responds", yesNo(res.Responds))
	if !res.Responds {
		if res.Note != "" {
			reportRow(stdout, "Note", res.Note)
		}
		return 0
	}
	reportRow(stdout, "Version", res.Version)
	reportRow(stdout, "NAT traversal", yesNo(res.NATTraversal))
	if res.RTTms > 0 {
		reportRow(stdout, "Round-trip", fmt.Sprintf("%d ms", res.RTTms))
	}
	for _, tr := range res.Accepted {
		reportRow(stdout, tr.Kind, tr.Name)
	}
	if len(res.Notify) > 0 {
		reportRow(stdout, "Notify", strings.Join(res.Notify, ", "))
	}
	if res.Note != "" {
		reportRow(stdout, "Note", res.Note)
	}
	return 0
}
