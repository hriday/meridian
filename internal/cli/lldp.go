package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/hriday/meridian/internal/lldp"
)

func runLLDP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lldp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	window := fs.Duration("wait", lldp.DefaultWindow, "how long to listen for advertisements")
	iface := fs.String("iface", "", "interface to listen on (default: the egress interface)")
	asJSON := fs.Bool("json", false, "output neighbors as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if !lldp.Supported() {
		fmt.Fprintln(stderr, "LLDP/CDP capture isn't supported on this operating system yet")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Fprintf(stderr, "Listening for LLDP/CDP for up to %s (Ctrl-C to stop early)…\n", *window)
	neighbors, err := lldp.Listen(ctx, *iface, *window)
	if err != nil {
		if errors.Is(err, lldp.ErrNeedsPrivilege) {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, neighbors)
	}
	printLLDP(stdout, neighbors)
	return 0
}

func printLLDP(w io.Writer, ns []lldp.Neighbor) {
	if len(ns) == 0 {
		fmt.Fprintln(w, "No LLDP/CDP neighbors heard. The switch may not advertise them, or none was sent in the listen window.")
		return
	}
	fmt.Fprintf(w, "%d neighbor(s):\n", len(ns))
	for i, n := range ns {
		if i > 0 {
			fmt.Fprintln(w)
		}
		reportRow(w, "Protocol", n.Protocol)
		if n.SystemName != "" {
			reportRow(w, "  Switch", n.SystemName)
		}
		if n.PortID != "" {
			port := n.PortID
			if n.PortDesc != "" && n.PortDesc != n.PortID {
				port += " (" + n.PortDesc + ")"
			}
			reportRow(w, "  Port", port)
		}
		if n.Platform != "" {
			reportRow(w, "  Platform", n.Platform)
		}
		if n.VLAN > 0 {
			reportRow(w, "  VLAN", fmt.Sprintf("%d", n.VLAN))
		}
		if n.MgmtIP != "" {
			reportRow(w, "  Mgmt IP", n.MgmtIP)
		}
		if n.ChassisID != "" {
			reportRow(w, "  Chassis", n.ChassisID)
		}
		if len(n.Capabilities) > 0 {
			reportRow(w, "  Capabilities", strings.Join(n.Capabilities, ", "))
		}
		if n.SystemDesc != "" {
			reportRow(w, "  System", n.SystemDesc)
		}
	}
}
