package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/smtpprobe"
)

func runSMTP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("smtp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.String("port", "", "server port (default 25, or 465 with --tls)")
	tlsImplicit := fs.Bool("tls", false, "connect with implicit TLS (default port 465)")
	useMX := fs.Bool("mx", false, "treat the argument as a domain and probe its top MX")
	ehlo := fs.String("ehlo", "", "name to send in EHLO")
	asJSON := fs.Bool("json", false, "output the result as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "smtp needs exactly one host or domain\n%s", usage)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	target := pos[0]
	if *useMX {
		mx, err := smtpprobe.TopMX(ctx, target)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "MX for %s: %s\n", target, mx)
		target = mx
	}
	p := *port
	if p == "" {
		p = "25"
		if *tlsImplicit {
			p = "465"
		}
	}

	d := &net.Dialer{}
	res, err := smtpprobe.Probe(ctx, d.DialContext, target, p, *ehlo, *tlsImplicit)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printSMTP(stdout, res)
	return 0
}

func printSMTP(w io.Writer, r smtpprobe.Result) {
	reportRow(w, "Server", net.JoinHostPort(r.Host, r.Port))
	reportRow(w, "Banner", r.Banner)
	if r.Greeting != "" {
		reportRow(w, "Greeting", r.Greeting)
	}
	tlsState := "no"
	if r.ImplicitTLS {
		tlsState = "implicit (connected over TLS)"
	} else if r.StartTLS {
		tlsState = "STARTTLS offered"
	}
	reportRow(w, "TLS", tlsState)

	if r.TLS != nil {
		reportRow(w, "  Protocol", r.TLS.Version+" / "+r.TLS.Cipher)
		reportRow(w, "  Cert subject", r.TLS.Subject)
		reportRow(w, "  Cert issuer", r.TLS.Issuer)
		match := "mismatch!"
		if r.TLS.HostMatch {
			match = "valid for this host"
		}
		reportRow(w, "  Cert validity", fmt.Sprintf("%s — expires in %d days (%s)", match, r.TLS.DaysLeft, r.TLS.NotAfter.Format("2006-01-02")))
	}
	if len(r.AuthMechs) > 0 {
		reportRow(w, "Auth", strings.Join(r.AuthMechs, ", "))
	}
	if len(r.Capabilities) > 0 {
		reportRow(w, "Capabilities", strings.Join(r.Capabilities, ", "))
	}
}
