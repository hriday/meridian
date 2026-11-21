package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/ble"
)

func runBLE(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ble", flag.ContinueOnError)
	fs.SetOutput(stderr)
	secs := fs.Int("seconds", 6, "scan duration in seconds (max 30)")
	asJSON := fs.Bool("json", false, "output JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	dur := time.Duration(*secs) * time.Second
	if dur <= 0 {
		dur = 6 * time.Second
	}
	if dur > 30*time.Second {
		dur = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), dur+10*time.Second)
	defer cancel()
	survey, err := ble.Scan(ctx, dur)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, survey)
	}
	printBLE(stdout, survey)
	return 0
}

func printBLE(w io.Writer, s ble.Survey) {
	reportRow(w, "Scanned", fmt.Sprintf("%ds", s.DurationSec))
	reportRow(w, "Devices", fmt.Sprintf("%d (%d named)", s.Count, s.Named))
	if len(s.Devices) == 0 {
		fmt.Fprintln(w, "\nNo BLE devices advertising nearby.")
		return
	}
	fmt.Fprintln(w, "\nNearby BLE devices (strongest first):")
	fmt.Fprintf(w, "  %-5s %-24s %-22s %s\n", "RSSI", "Name", "Company", "Address")
	for _, d := range s.Devices {
		name := d.Name
		if name == "" {
			name = "(unnamed)"
		}
		fmt.Fprintf(w, "  %-5d %-24s %-22s %s\n", d.RSSI, truncateRunes(name, 24), truncateRunes(d.Company, 22), d.Address)
	}
}
