package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/hriday/meridian/internal/wgprobe"
)

func runWg(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wg", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 51820, "UDP port")
	pubkey := fs.String("pubkey", "", "peer (responder) public key, base64 — needed for a meaningful probe")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "wg needs a host (WireGuard endpoint)\n%s", usage)
		return 2
	}
	host := net.JoinHostPort(pos[0], strconv.Itoa(*port))

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := wgprobe.Probe(ctx, host, *pubkey)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	reportRow(stdout, "Target", res.Target)
	reportRow(stdout, "Valid mac1", yesNo(res.ValidMAC1))
	reportRow(stdout, "Responds", yesNo(res.Responds))
	if res.Responds {
		reportRow(stdout, "Reply", res.ReplyType)
		if res.RTTms > 0 {
			reportRow(stdout, "Round-trip", fmt.Sprintf("%d ms", res.RTTms))
		}
	}
	reportRow(stdout, "Verdict", res.Verdict)
	return 0
}
