package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hriday/meridian/internal/stun"
)

var defaultSTUN = []string{"stun.cloudflare.com:3478", "stun.l.google.com:19302"}

func runSTUN(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stun", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	servers := defaultSTUN
	if len(pos) >= 1 {
		s := pos[0]
		if _, _, err := net.SplitHostPort(s); err != nil {
			s = net.JoinHostPort(s, "3478")
		}
		servers = append([]string{s}, defaultSTUN...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := stun.Discover(ctx, servers)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printSTUN(stdout, res)
	return 0
}

func printSTUN(w io.Writer, r stun.Result) {
	reportRow(w, "STUN server", fmt.Sprintf("%s  (%dms)", r.Server, r.RTTMillis))
	reportRow(w, "Local IP", r.LocalIP)
	reportRow(w, "Public (reflexive)", fmt.Sprintf("%s:%d", r.PublicIP, r.PublicPort))
	reportRow(w, "NAT", map[bool]string{true: "yes", false: "no"}[r.NAT])
	reportRow(w, "Assessment", r.NATType)
}
