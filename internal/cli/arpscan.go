package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/hriday/meridian/internal/arpscan"
	"github.com/hriday/meridian/internal/l2"
)

func runARP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("arpscan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	window := fs.Duration("wait", arpscan.DefaultWindow, "how long to listen for replies after the sweep")
	max := fs.Int("max", arpscan.DefaultMaxHosts, "maximum hosts to probe")
	noVendor := fs.Bool("no-vendor", false, "skip the online MAC-vendor lookup")
	asJSON := fs.Bool("json", false, "output hosts as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if !arpscan.Supported() {
		fmt.Fprintln(stderr, "ARP scanning isn't supported on this operating system yet")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Fprintln(stderr, "Sweeping the local subnet with ARP…")
	res, err := arpscan.Run(ctx, *window, *max)
	if err != nil {
		if errors.Is(err, l2.ErrNeedsPrivilege) {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !*noVendor {
		arpscan.LookupVendors(ctx, &http.Client{Timeout: 8 * time.Second}, res.Hosts)
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printARP(stdout, res)
	return 0
}

func printARP(w io.Writer, r arpscan.Result) {
	fmt.Fprintf(w, "%d host(s) on %s (%s):\n\n", len(r.Hosts), r.Subnet, r.Interface)
	if len(r.Hosts) == 0 {
		return
	}
	fmt.Fprintf(w, "%-16s %-18s %s\n", "IP", "MAC", "Vendor")
	for _, h := range r.Hosts {
		fmt.Fprintf(w, "%-16s %-18s %s\n", h.IP, h.MAC, h.Vendor)
	}
}
