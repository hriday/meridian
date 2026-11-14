package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/certinfo"
)

func runCertInfo(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("certinfo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	data, err := readFileOrStdin(pos, os.Stdin)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	res, err := certinfo.Decode(string(data), time.Now())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printCertInfo(stdout, res)
	return 0
}

func printCertInfo(w io.Writer, res certinfo.Result) {
	for i, it := range res.Items {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if it.Kind == "csr" {
			fmt.Fprintf(w, "Certificate Signing Request\n")
		} else {
			fmt.Fprintf(w, "Certificate\n")
		}
		reportRow(w, "Subject", it.Subject)
		if it.Issuer != "" {
			tag := it.Issuer
			if it.SelfSigned {
				tag += "  (self-signed)"
			}
			reportRow(w, "Issuer", tag)
		}
		reportRow(w, "Key", it.KeyType)
		reportRow(w, "Signature", it.SigAlg)
		if it.SerialHex != "" {
			reportRow(w, "Serial", it.SerialHex)
		}
		if it.NotAfter != nil {
			status := "valid"
			if it.Expired {
				status = "EXPIRED"
			}
			reportRow(w, "Valid until", fmt.Sprintf("%s (%d days, %s)", it.NotAfter.Format("2006-01-02"), it.DaysLeft, status))
		}
		if it.IsCA {
			reportRow(w, "CA", "yes")
		}
		if len(it.DNSNames) > 0 {
			reportRow(w, "DNS names", strings.Join(it.DNSNames, ", "))
		}
		if len(it.IPs) > 0 {
			reportRow(w, "IP SANs", strings.Join(it.IPs, ", "))
		}
		if len(it.Emails) > 0 {
			reportRow(w, "Emails", strings.Join(it.Emails, ", "))
		}
		if len(it.URIs) > 0 {
			reportRow(w, "URIs", strings.Join(it.URIs, ", "))
		}
		if len(it.KeyUsage) > 0 {
			reportRow(w, "Key usage", strings.Join(it.KeyUsage, ", "))
		}
		if len(it.ExtKeyUsage) > 0 {
			reportRow(w, "Ext key usage", strings.Join(it.ExtKeyUsage, ", "))
		}
	}
}
