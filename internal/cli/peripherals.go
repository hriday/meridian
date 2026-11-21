package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/peripherals"
)

func runPeripherals(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("peripherals", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	survey, err := peripherals.Scan(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, survey)
	}
	printPeripherals(stdout, survey)
	return 0
}

func printPeripherals(w io.Writer, s peripherals.Survey) {
	reportRow(w, "Attached", fmt.Sprintf("%d device(s), %d flagged", s.Count, s.Flagged))
	if s.Note != "" {
		reportRow(w, "Note", s.Note)
	}
	if len(s.Devices) == 0 {
		fmt.Fprintln(w, "\nNo USB peripherals attached. (This can't see wireless devices in the air — a 2.4 GHz dongle only appears once it's plugged in.)")
		return
	}
	fmt.Fprintln(w, "\nAttached peripherals (anything flagged first):")
	for _, d := range s.Devices {
		vendor := d.Vendor
		if vendor == "" {
			vendor = "unknown vendor"
		}
		ids := ""
		if d.VendorID != "" {
			ids = fmt.Sprintf("  [%s:%s]", d.VendorID, d.ProductID)
		}
		fmt.Fprintf(w, "\n  %-20s %s — %s%s\n", d.Kind, d.Name, vendor, ids)
		for _, f := range d.Flags {
			mark := "·"
			if f.Level == "watch" {
				mark = "!"
			}
			fmt.Fprintf(w, "    %s %s\n", mark, f.Text)
		}
	}
	fmt.Fprintln(w, "\nNo flags isn't a clean bill of health — a malicious device can look ordinary. Point-in-time; Meridian keeps no history.")
}
