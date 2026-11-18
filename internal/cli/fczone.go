package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/fczone"
)

// fcStdin is the config source when no file argument is given; tests override it.
var fcStdin io.Reader = os.Stdin

func runFCZone(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fczone", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	var data []byte
	var err error
	if len(pos) == 1 {
		data, err = os.ReadFile(pos[0])
	} else {
		data, err = io.ReadAll(fcStdin)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if strings.TrimSpace(string(data)) == "" {
		fmt.Fprintf(stderr, "fczone needs a switch zoning config: a file arg, or pipe `show zoneset` / `cfgshow` on stdin\n%s", usage)
		return 2
	}
	r := fczone.Parse(string(data))
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	printFCZone(stdout, r)
	return 0
}

func printFCZone(w io.Writer, r fczone.Report) {
	reportRow(w, "Vendor", r.Vendor)
	reportRow(w, "Summary", fmt.Sprintf("%d zoneset(s) · %d zone(s) · %d alias(es)", r.Counts.Zonesets, r.Counts.Zones, r.Counts.Aliases))
	for i, zs := range r.Zonesets {
		label := "Zonesets"
		if i > 0 {
			label = ""
		}
		mark := ""
		if zs.Active {
			mark = "  [active]"
		}
		reportRow(w, label, fmt.Sprintf("%s — %d zone(s)%s", zs.Name, len(zs.Zones), mark))
	}
	if len(r.Findings) > 0 {
		fmt.Fprintln(w, "\nFindings:")
		for _, f := range r.Findings {
			obj := ""
			if f.Object != "" {
				obj = " [" + f.Object + "]"
			}
			fmt.Fprintf(w, "  [%-7s] %s%s\n      %s\n", strings.ToUpper(f.Severity), f.Title, obj, f.Detail)
		}
	} else {
		fmt.Fprintln(w, "\nNo hygiene findings.")
	}
	if r.Counts.Unparsed > 0 {
		fmt.Fprintf(w, "\nNote: %d line(s) weren't recognized (coverage gap) — the parse may be incomplete.\n", r.Counts.Unparsed)
	}
}
