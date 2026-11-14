package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hriday/meridian/internal/acmecert"
)

// certStdin is where the "press Enter" confirmation is read; tests override it.
var certStdin io.Reader = os.Stdin

func runCertIssue(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("certissue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	email := fs.String("email", "", "contact email (optional; for expiry notices)")
	staging := fs.Bool("staging", false, "use Let's Encrypt staging (untrusted test cert)")
	out := fs.String("out", ".", "directory to write fullchain.pem and privkey.pem")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) == 0 {
		fmt.Fprintf(stderr, "certissue needs at least one domain\n%s", usage)
		return 2
	}

	beginCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	order, records, err := acmecert.Begin(beginCtx, acmecert.Directory(*staging), *email, pos)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(records) == 0 {
		fmt.Fprintln(stderr, "nothing to validate (domains may already be authorized)")
		return 1
	}

	fmt.Fprintln(stdout, "Publish these TXT record(s) at your DNS:")
	for _, r := range records {
		fmt.Fprintf(stdout, "  %-40s TXT  %s\n", r.Name, r.Value)
	}
	fmt.Fprint(stdout, "\nPress Enter once they're published and have propagated… ")
	_, _ = bufio.NewReader(certStdin).ReadString('\n')

	finCtx, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel2()
	chain, key, err := order.Finalize(finCtx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	chainPath := filepath.Join(*out, "fullchain.pem")
	keyPath := filepath.Join(*out, "privkey.pem")
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(chainPath, []byte(chain), 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(keyPath, []byte(key), 0o600); err != nil { // 0600: private key
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "\nIssued for %v\n  %s\n  %s (mode 0600 — keep it secret)\n", order.Domains(), chainPath, keyPath)
	return 0
}
