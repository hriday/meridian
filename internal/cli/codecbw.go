package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/hriday/meridian/internal/voipcalc"
)

func runCodecBW(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("codecbw", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ptime := fs.Int("ptime", 0, "packetization interval in ms (0 = codec default)")
	calls := fs.Int("calls", 1, "number of concurrent calls")
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		var names []string
		for _, c := range voipcalc.Codecs() {
			names = append(names, firstToken(c.Name))
		}
		fmt.Fprintf(stderr, "codecbw needs a codec: %s\n", strings.Join(names, ", "))
		return 2
	}
	r, err := voipcalc.Calc(pos[0], *ptime, *calls)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	printCodecBW(stdout, r)
	return 0
}

func printCodecBW(w io.Writer, r voipcalc.Result) {
	reportRow(w, "Codec", fmt.Sprintf("%s @ %dms", r.Codec, r.PtimeMs))
	reportRow(w, "Payload", fmt.Sprintf("%d bytes/packet, %.0f packets/s", r.PayloadBytes, r.PacketsPerSec))
	reportRow(w, "One-way (IP)", fmt.Sprintf("%.1f kbps", r.OneWayKbps))
	reportRow(w, "One-way (Ethernet)", fmt.Sprintf("%.1f kbps", r.OneWayEthKbps))
	reportRow(w, "Per call (2-way)", fmt.Sprintf("%.1f kbps", r.PerCallKbps))
	reportRow(w, fmt.Sprintf("%d calls (IP)", r.Calls), fmt.Sprintf("%.1f kbps  (%.2f Mbps)", r.TotalKbps, r.TotalKbps/1000))
	reportRow(w, fmt.Sprintf("%d calls (Ethernet)", r.Calls), fmt.Sprintf("%.1f kbps  (%.2f Mbps)", r.TotalEthKbps, r.TotalEthKbps/1000))
}

func firstToken(s string) string {
	if i := strings.IndexAny(s, " ("); i > 0 {
		return s[:i]
	}
	return s
}
