package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/wifi"
)

func runWifi(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wifi", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if !wifi.Supported() {
		fmt.Fprintln(stderr, "Wi-Fi analysis isn't supported on this operating system yet (macOS only)")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	info, err := wifi.Status(ctx)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, info)
	}

	c := info.Current
	reportRow(stdout, "Interface", info.Interface)
	reportRow(stdout, "Network", fmt.Sprintf("%s  (ch %d · %s · %s)", c.SSID, c.Channel, c.Band, c.Width))
	reportRow(stdout, "PHY / Sec", c.PHYMode+" · "+c.Security)
	reportRow(stdout, "Signal", fmt.Sprintf("%d dBm · noise %d dBm · SNR %d dB · %d Mbps", c.RSSI, c.Noise, c.SNR, c.TxRate))
	if info.CoChannel > 0 {
		reportRow(stdout, "Co-channel", fmt.Sprintf("%d neighbor(s) also on channel %d", info.CoChannel, c.Channel))
	}

	if len(info.Neighbors) > 0 {
		fmt.Fprintf(stdout, "\nNeighbors (%d):\n", len(info.Neighbors))
		for _, n := range info.Neighbors {
			co := ""
			if n.CoChannel {
				co = "  [co-channel]"
			}
			fmt.Fprintf(stdout, "  ch %-4d %-5s %4d dBm  %s%s\n", n.Channel, n.Band, n.RSSI, n.SSID, co)
		}
	}
	if info.Note != "" {
		fmt.Fprintf(stdout, "\n%s\n", info.Note)
	}
	return 0
}
