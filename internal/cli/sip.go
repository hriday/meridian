package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/sipping"
)

func runSIP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sip", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.String("port", "", "SIP port (default 5060, or 5061 for TLS)")
	transport := fs.String("transport", "udp", "udp | tcp | tls")
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "sip needs exactly one host\n%s", usage)
		return 2
	}
	p := *port
	if p == "" {
		p = "5060"
		if *transport == "tls" {
			p = "5061"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	d := &net.Dialer{}
	res, err := sipping.Probe(ctx, d.DialContext, pos[0], p, *transport)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printSIP(stdout, res)
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return 0
	}
	return 0 // any SIP reply proves reachability; non-2xx still printed
}

func printSIP(w io.Writer, r sipping.Result) {
	reportRow(w, "Server", fmt.Sprintf("%s  (%s, %dms)", r.Target, strings.ToUpper(r.Transport), r.RTTMillis))
	verdict := fmt.Sprintf("%d %s", r.StatusCode, r.Reason)
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		verdict += "  — reachable and healthy"
	} else if r.StatusCode > 0 {
		verdict += "  — reachable (SIP is answering)"
	}
	reportRow(w, "Response", verdict)
	if r.Server != "" {
		reportRow(w, "Server header", r.Server)
	}
	if r.UserAgent != "" {
		reportRow(w, "User-Agent", r.UserAgent)
	}
	if len(r.Allow) > 0 {
		reportRow(w, "Methods (Allow)", strings.Join(r.Allow, ", "))
	}
	if len(r.Supported) > 0 {
		reportRow(w, "Supported", strings.Join(r.Supported, ", "))
	}
	if len(r.Accept) > 0 {
		reportRow(w, "Accept", strings.Join(r.Accept, ", "))
	}
}
