package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/pktscan"
)

func runPktScan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pktscan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	iface := fs.String("iface", "", "interface to capture on (default: egress)")
	seconds := fs.Int("seconds", int(pktscan.DefaultWindow/time.Second), "capture window in seconds")
	top := fs.Int("top", 10, "number of top talkers to show")
	asJSON := fs.Bool("json", false, "output as JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if !pktscan.Supported() {
		fmt.Fprintln(stderr, "packet capture isn't supported on this operating system yet")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Fprintf(stderr, "Capturing for %ds… (Ctrl-C to stop early)\n", *seconds)
	res, err := pktscan.Run(ctx, *iface, time.Duration(*seconds)*time.Second, *top)
	if err != nil {
		fmt.Fprintln(stderr, err)
		if res.Packets == 0 {
			return 1
		}
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printPktScan(stdout, res)
	return 0
}

func printPktScan(w io.Writer, r pktscan.Summary) {
	fmt.Fprintf(w, "Captured %d packets (%s) on %s in %.1fs\n\n", r.Packets, humanBytes(r.Bytes), r.Interface, r.Seconds)
	if r.Packets == 0 {
		fmt.Fprintln(w, "No packets. On a switched LAN you only see traffic you're in the path of (your own, broadcast/multicast, or a SPAN/mirror port).")
		return
	}

	var protos []string
	for _, p := range r.Protocols {
		protos = append(protos, fmt.Sprintf("%s %d", p.Proto, p.Packets))
	}
	fmt.Fprintf(w, "Protocols: %s\n", strings.Join(protos, "  ·  "))
	fmt.Fprintf(w, "TCP: %d SYN · %d RST · %d retransmit(s)\n\n", r.TCPSyns, r.TCPResets, r.Retransmits)

	fmt.Fprintln(w, "Top talkers:")
	for _, t := range r.TopTalkers {
		fmt.Fprintf(w, "  %-24s %9s  %d pkts\n", t.IP, humanBytes(t.Bytes), t.Packets)
	}
	if len(r.DNSErrors) > 0 {
		fmt.Fprintln(w, "\nDNS errors:")
		for _, d := range r.DNSErrors {
			fmt.Fprintf(w, "  %-10s %d\n", d.Name, d.Count)
		}
	}
	if len(r.SNIs) > 0 {
		fmt.Fprintf(w, "\nTLS SNIs (%d): %s\n", len(r.SNIs), strings.Join(r.SNIs, ", "))
	}
}

func humanBytes(n int) string {
	const u = 1024
	if n < u {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(u), 0
	for v := int64(n) / u; v >= u; v /= u {
		div *= u
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
