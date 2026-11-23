package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/meshtastic"
)

func runMeshtastic(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mesh", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) == 1 && pos[0] == "ports" {
		return runMeshtasticPorts(*asJSON, stdout, stderr)
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "mesh needs a Meshtastic node: a serial device (e.g. /dev/cu.usbserial-10) or a host (e.g. 192.168.1.50), or 'ports' to list attached USB serial devices\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	snap, err := meshtastic.Read(ctx, pos[0])
	if err != nil {
		// Report the failure on stderr and exit non-zero even in --json mode —
		// never print a zero-value snapshot to stdout and claim success.
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, snap)
	}
	printMeshtastic(stdout, snap)
	return 0
}

// runMeshtasticPorts lists the USB serial devices a node could be reached on,
// so a serial target can be found without hunting for its device path.
func runMeshtasticPorts(asJSON bool, stdout, stderr io.Writer) int {
	ports, err := meshtastic.SerialPorts()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if asJSON {
		return writeJSON(stdout, stderr, ports)
	}
	if len(ports) == 0 {
		fmt.Fprintln(stdout, "No USB serial devices attached. Plug the node in over USB, or connect over TCP instead.")
		return 0
	}
	fmt.Fprintln(stdout, "Attached USB serial devices (pass one as the node):")
	for _, p := range ports {
		fmt.Fprintf(stdout, "  %-28s %s\n", p.Path, p.Label)
	}
	return 0
}

func printMeshtastic(w io.Writer, s meshtastic.Snapshot) {
	reportRow(w, "Transport", fmt.Sprintf("%s (%s)", s.Transport, s.Target))
	if s.Me != nil {
		name := s.Me.LongName
		if name == "" {
			name = s.Me.ID
		}
		reportRow(w, "This node", fmt.Sprintf("%s  %s", name, s.Me.ID))
	}
	if s.HwModel != "" {
		reportRow(w, "Hardware", s.HwModel)
	}
	if s.Firmware != "" {
		reportRow(w, "Firmware", s.Firmware)
	}
	reportRow(w, "Channel util", fmt.Sprintf("%.1f%%  (tx airtime %.1f%%)", s.ChannelUtil, s.AirUtilTx))
	reportRow(w, "Nodes seen", fmt.Sprintf("%d", s.NodeCount))
	if r := s.Radio; r != nil {
		region := r.Region
		if !r.RegionSet {
			region = "UNSET"
		}
		preset := r.ModemPreset
		if preset == "" {
			preset = "custom"
		}
		ch := r.PrimaryChannel
		if ch == "" {
			ch = "(default public)"
		}
		tx := "on"
		if !r.TxEnabled {
			tx = "OFF"
		}
		reportRow(w, "Radio", fmt.Sprintf("region %s · preset %s · channel %s · tx %s", region, preset, ch, tx))
	}

	if len(s.Findings) > 0 {
		fmt.Fprintln(w, "\nMesh health:")
		for _, f := range s.Findings {
			marker := "·"
			if f.Level == "problem" {
				marker = "!"
			}
			fmt.Fprintf(w, "  %s %s\n", marker, f.Title)
			if f.Fix != "" {
				fmt.Fprintf(w, "    → %s\n", f.Fix)
			}
		}
	}

	if len(s.Nodes) == 0 {
		fmt.Fprintln(w, "\nNo other nodes in the database.")
		return
	}
	fmt.Fprintln(w, "\nMesh nodes (nearest first):")
	fmt.Fprintf(w, "  %-11s %-20s %-5s %-6s %-6s %s\n", "ID", "Name", "Hops", "SNR", "Heard", "Position")
	for _, n := range s.Nodes {
		hops := "?"
		switch {
		case n.ViaMQTT:
			hops = "mqtt"
		case n.HopsAway != nil:
			hops = fmt.Sprintf("%d", *n.HopsAway)
		}
		name := n.LongName
		if name == "" {
			name = n.ShortName
		}
		pos := ""
		if n.HasPosition {
			pos = fmt.Sprintf("%.4f,%.4f", n.Lat, n.Lon)
		}
		fmt.Fprintf(w, "  %-11s %-20s %-5s %-6.1f %-6s %s\n",
			n.ID, truncateRunes(name, 20), hops, n.SNR, relHeard(n.LastHeard), pos)
	}
}

// relHeard renders an epoch-seconds last-heard time as a compact relative age.
func relHeard(epoch int64) string {
	if epoch <= 0 {
		return "—"
	}
	d := time.Since(time.Unix(epoch, 0))
	switch {
	case d < 0:
		return "now"
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
