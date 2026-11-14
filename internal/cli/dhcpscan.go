package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/dhcpscan"
)

func runDHCP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dhcpdiscover", flag.ContinueOnError)
	fs.SetOutput(stderr)
	window := fs.Duration("wait", dhcpscan.DefaultWindow, "how long to listen for offers")
	asJSON := fs.Bool("json", false, "output the result as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if !dhcpscan.Supported() {
		fmt.Fprintln(stderr, "DHCP discovery isn't supported on this operating system yet")
		return 1
	}

	res, err := dhcpscan.Run(*window)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printDHCP(stdout, res)
	return 0
}

func printDHCP(w io.Writer, r dhcpscan.Result) {
	if len(r.Servers) == 0 {
		fmt.Fprintf(w, "No DHCP server answered on %s within the listen window.\n", r.Interface)
		return
	}
	if r.Rogue {
		fmt.Fprintf(w, "⚠ ROGUE: %d DHCP servers answered on %s — only one should.\n\n", len(r.Servers), r.Interface)
	} else {
		fmt.Fprintf(w, "1 DHCP server answered on %s.\n\n", r.Interface)
	}
	for i, s := range r.Servers {
		if i > 0 {
			fmt.Fprintln(w)
		}
		reportRow(w, "Server", s.ServerID)
		reportRow(w, "  Offered address", s.Offered)
		if s.SubnetMask != "" {
			reportRow(w, "  Subnet mask", s.SubnetMask)
		}
		if s.Gateway != "" {
			reportRow(w, "  Gateway", s.Gateway)
		}
		if len(s.DNS) > 0 {
			reportRow(w, "  DNS", strings.Join(s.DNS, ", "))
		}
		if s.DomainName != "" {
			reportRow(w, "  Domain", s.DomainName)
		}
		if s.LeaseSecs > 0 {
			reportRow(w, "  Lease", (time.Duration(s.LeaseSecs) * time.Second).String())
		}
	}
}
