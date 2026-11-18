package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/myip"
	"github.com/hriday/meridian/internal/stun"
)

func runMyIP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("myip", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output as JSON")
	localOnly := fs.Bool("local", false, "skip the public-IP lookup (no external STUN query)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 0 {
		fmt.Fprintf(stderr, "myip takes no arguments\n%s", usage)
		return 2
	}

	local, err := myip.Local()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	var pub stun.Result
	var pubErr error
	if !*localOnly {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		pub, pubErr = stun.Discover(ctx, defaultSTUN)
	}

	if *asJSON {
		out := struct {
			myip.Info
			PublicIP    string `json:"publicIP,omitempty"`
			NATType     string `json:"natType,omitempty"`
			PublicError string `json:"publicError,omitempty"`
		}{Info: local}
		switch {
		case *localOnly:
		case pubErr != nil:
			out.PublicError = pubErr.Error()
		default:
			out.PublicIP = pub.PublicIP
			out.NATType = pub.NATType
		}
		return writeJSON(stdout, stderr, out)
	}

	if local.PrimaryIP != "" {
		row := local.PrimaryIP
		if local.PrimaryIf != "" {
			row += "  (" + local.PrimaryIf + ")"
		}
		reportRow(stdout, "Local IP", row)
	}
	if local.Gateway != "" {
		reportRow(stdout, "Gateway", local.Gateway)
	}
	if !*localOnly {
		if pubErr != nil {
			// A failed lookup is reported as unavailable — never a fake 0/absent.
			reportRow(stdout, "Public IP", "unavailable ("+pubErr.Error()+")")
		} else {
			reportRow(stdout, "Public IP", pub.PublicIP)
			if pub.NATType != "" {
				reportRow(stdout, "NAT", pub.NATType)
			}
		}
	}
	if len(local.Interfaces) > 0 {
		fmt.Fprintln(stdout, "Interfaces:")
		for _, f := range local.Interfaces {
			reportRow(stdout, "  "+f.Name, strings.Join(f.IPv4, ", "))
		}
	}
	return 0
}
