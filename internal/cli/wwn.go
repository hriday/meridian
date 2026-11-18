package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/macvendor"
	"github.com/hriday/meridian/internal/wwn"
)

func runWWN(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wwn", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "wwn needs one WWN/WWPN, e.g. 20:00:00:25:b5:00:00:01\n%s", usage)
		return 2
	}
	r, err := wwn.Parse(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if r.OUIBare != "" && r.OUIBare != "000000" {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if mv, mErr := macvendor.Lookup(ctx, &http.Client{Timeout: 5 * time.Second}, r.OUIBare); mErr == nil && mv.Found {
			r.Vendor = mv.Vendor
		}
	}
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	printWWN(stdout, r)
	return 0
}

func printWWN(w io.Writer, r wwn.Result) {
	reportRow(w, "WWN", r.Canonical)
	reportRow(w, "Cisco form", r.Dotted)
	reportRow(w, "Format", r.NAADesc)
	if r.OUI != "" {
		v := r.OUI
		if r.Vendor != "" {
			v += "  —  " + r.Vendor
		}
		reportRow(w, "OUI / vendor", v)
	}
	if r.VSID != "" {
		reportRow(w, "Vendor-specific", r.VSID)
	}
	for i, n := range r.Notes {
		label := "Notes"
		if i > 0 {
			label = ""
		}
		reportRow(w, label, n)
	}
}
