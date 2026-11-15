package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/discover"
)

func runDiscover(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	fs.SetOutput(stderr)
	seconds := fs.Int("seconds", 8, "total listening window in seconds")
	asJSON := fs.Bool("json", false, "output as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if *seconds < 1 {
		*seconds = 8
	}
	if *seconds > 25 {
		*seconds = 25
	}

	window := time.Duration(*seconds) * time.Second
	// mDNS holds one socket open for the whole window (a cold unicast answer can
	// take ~4-5s); give the context a little headroom beyond it.
	ctx, cancel := context.WithTimeout(context.Background(), window+5*time.Second)
	defer cancel()
	res, err := discover.Run(ctx, window)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printDiscover(stdout, res)
	return 0
}

func printDiscover(w io.Writer, r discover.Result) {
	fmt.Fprintf(w, "mDNS / Bonjour — %d service(s)\n", len(r.Services))
	for _, s := range r.Services {
		name := s.Instance
		if name == "" {
			name = s.Host
		}
		addr := s.Host
		if len(s.Addrs) > 0 {
			addr = strings.Join(s.Addrs, ", ")
		}
		fmt.Fprintf(w, "  %-22s %-24s %s:%d\n", trunc(name, 22), s.Type, addr, s.Port)
		for _, t := range s.Info {
			fmt.Fprintf(w, "      %s\n", t)
		}
	}
	fmt.Fprintf(w, "\nSSDP / UPnP — %d device(s)\n", len(r.Devices))
	for _, d := range r.Devices {
		fmt.Fprintf(w, "  %-15s %s\n", d.Address, d.Type)
		if d.Server != "" {
			fmt.Fprintf(w, "      server:   %s\n", d.Server)
		}
		if d.Location != "" {
			fmt.Fprintf(w, "      location: %s\n", d.Location)
		}
	}
	if len(r.Services) == 0 && len(r.Devices) == 0 {
		fmt.Fprintln(w, "\nNothing answered. Devices advertise only on their own link — run this on the same LAN segment (not over a VPN or a routed/guest network that blocks multicast).")
	}
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}
