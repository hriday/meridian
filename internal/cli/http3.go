package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/h3probe"
)

func runHTTP3(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("http3", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 443, "UDP port to probe")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "http3 needs a host\n%s", usage)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	res := h3probe.Probe(ctx, pos[0], *port)

	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	reportRow(stdout, "Host", fmt.Sprintf("%s:%d", res.Host, res.Port))
	reportRow(stdout, "QUIC reachable", yesNo(res.QUICReachable))
	if res.QUICReachable {
		reportRow(stdout, "ALPN", res.ALPN)
		reportRow(stdout, "QUIC version", res.QUICVersion)
		reportRow(stdout, "TLS", res.TLSVersion)
		reportRow(stdout, "Handshake", fmt.Sprintf("%d ms", res.HandshakeMillis))
	} else if res.QUICError != "" {
		reportRow(stdout, "QUIC error", res.QUICError)
	}
	reportRow(stdout, "Alt-Svc h3", yesNo(res.AdvertisesH3))
	if res.AltSvc != "" {
		reportRow(stdout, "Alt-Svc", res.AltSvc)
	}
	if h := res.HTTPS; h != nil && h.Found {
		reportRow(stdout, "HTTPS record", "advertises "+strings.Join(h.ALPN, ", "))
		reportRow(stdout, "  h3 via DNS", yesNo(h.AdvertisesH3))
		reportRow(stdout, "  ECH", yesNo(h.ECH))
		if h.Port != 0 {
			reportRow(stdout, "  port", strconv.Itoa(h.Port))
		}
		if hints := append(append([]string{}, h.IPv4Hint...), h.IPv6Hint...); len(hints) > 0 {
			reportRow(stdout, "  hints", strings.Join(hints, ", "))
		}
	} else if h != nil {
		reportRow(stdout, "HTTPS record", "none published")
	}
	fmt.Fprintln(stdout, "\n"+res.Verdict)
	return 0
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
