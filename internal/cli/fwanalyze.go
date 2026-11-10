package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/fwconfig"
)

// fwStdin is the config source when no file argument is given; tests override it.
var fwStdin io.Reader = os.Stdin

func runFwAnalyze(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fwanalyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}

	var data []byte
	var err error
	if len(pos) >= 1 {
		data, err = os.ReadFile(pos[0])
	} else {
		data, err = io.ReadAll(fwStdin)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	analyses, err := fwconfig.Analyze(string(data), "fortigate")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, analyses)
	}
	printFwSummary(stdout, analyses)
	return 0
}

func printFwSummary(stdout io.Writer, analyses []fwconfig.Analysis) {
	for _, a := range analyses {
		name := a.VDOM
		if name == "" {
			name = "(single VDOM)"
		}
		fmt.Fprintf(stdout, "VDOM %s — %d VLANs, %d policies, %d addresses, %d services, %d zones, %d subnets/IPs\n",
			name, a.Counts.VLANs, a.Counts.Policies, a.Counts.Addresses, a.Counts.Services, a.Counts.Zones, a.Counts.SubnetsIPs)
		if len(a.Flows) > 0 {
			fmt.Fprintln(stdout, "  what talks to what:")
			for _, f := range a.Flows {
				svc := strings.Join(f.Services, ", ")
				if svc == "" {
					svc = "any"
				}
				fmt.Fprintf(stdout, "    %s → %s  [%s]  %s\n", f.Src, f.Dst, f.Action, svc)
			}
		}
		for _, n := range a.Notes {
			fmt.Fprintf(stdout, "  note: %s\n", n)
		}
		fmt.Fprintln(stdout)
	}
}
