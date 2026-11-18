package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/ipp"
)

func runPrinter(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("printer", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "printer needs a printer IP or hostname, e.g. 10.0.0.50\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
	defer cancel()
	r, err := ipp.Probe(ctx, &http.Client{Timeout: 8 * time.Second}, pos[0])
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	printPrinter(stdout, r)
	if err != nil && !r.Reachable {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func printPrinter(w io.Writer, r ipp.Result) {
	reportRow(w, "Host", r.Host)
	if r.Reachable {
		model := r.MakeModel
		if model == "" {
			model = r.Name
		}
		reportRow(w, "Printer", model)
		if r.Location != "" {
			reportRow(w, "Location", r.Location)
		}
		reportRow(w, "State", r.State)
		if len(r.StateReasons) > 0 {
			reportRow(w, "Attention", strings.Join(r.StateReasons, ", "))
		}
		for i, s := range r.Supplies {
			label := "Supplies"
			if i > 0 {
				label = ""
			}
			reportRow(w, label, fmt.Sprintf("%-18s %s", s.Name, supplyPct(s)))
		}
		reportRow(w, "Queued jobs", fmt.Sprintf("%d", r.QueuedJobs))
	} else if r.Note != "" {
		reportRow(w, "Note", r.Note)
	}
	fmt.Fprintln(w, "\nPrint ports:")
	for _, p := range r.Ports {
		state := "closed"
		if p.Open {
			state = "open"
		}
		fmt.Fprintf(w, "  %-5d %-16s %s\n", p.Port, p.Name, state)
	}
}

// supplyPct renders a marker level as a percentage where possible.
func supplyPct(s ipp.Supply) string {
	if s.Level < 0 {
		return "level unknown"
	}
	max := s.Max
	if max == 0 {
		max = 100
	}
	pct := s.Level
	if max != 100 {
		pct = s.Level * 100 / max
	}
	out := fmt.Sprintf("%d%%", pct)
	if s.Color != "" {
		out += "  (" + s.Color + ")"
	}
	return out
}
