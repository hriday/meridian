package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/sshaudit"
)

func runSSHAudit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sshaudit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) < 1 {
		fmt.Fprintln(stderr, "usage: sshaudit <host[:port]> [--json]")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r, err := sshaudit.Probe(ctx, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	printSSHAudit(stdout, r)
	return 0
}

func printSSHAudit(w io.Writer, r sshaudit.Report) {
	reportRow(w, "Host", r.Host)
	reportRow(w, "Software", r.Software)
	reportRow(w, "Grade", r.Grade)
	reportRow(w, "PQ key exchange", r.PQ.Verdict)
	for i, hk := range r.HostKeys {
		label := "Host keys"
		if i > 0 {
			label = ""
		}
		bits := ""
		if hk.Bits > 0 {
			bits = fmt.Sprintf(" (%d-bit)", hk.Bits)
		}
		reportRow(w, label, fmt.Sprintf("%s%s  %s", hk.Type, bits, hk.Fingerprint))
	}
	reportRow(w, "Key exchange", strings.Join(r.Kex, ", "))
	reportRow(w, "Ciphers", strings.Join(r.Ciphers, ", "))
	reportRow(w, "MACs", strings.Join(r.MACs, ", "))
	for i, f := range r.Findings {
		label := "Findings"
		if i > 0 {
			label = ""
		}
		reportRow(w, label, f)
	}
	if len(r.Remediation) > 0 {
		fmt.Fprintf(w, "\nHow to improve (toward grade A):\n")
		for _, s := range r.Remediation {
			fmt.Fprintf(w, "  • %s\n", s)
		}
	}
}
