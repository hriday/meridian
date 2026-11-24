package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/meshcore"
)

func runMeshCore(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("meshcore", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	// Serial devices are serial devices regardless of firmware, so the port
	// listing is shared with the Meshtastic tool.
	if len(pos) == 1 && pos[0] == "ports" {
		return runMeshtasticPorts(*asJSON, stdout, stderr)
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "meshcore needs a MeshCore node: a serial device (e.g. /dev/cu.usbmodem1101) or a host (e.g. 192.168.1.50), or 'ports' to list attached USB serial devices\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	snap, err := meshcore.Read(ctx, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, snap)
	}
	printMeshCore(stdout, snap)
	return 0
}

func printMeshCore(w io.Writer, s meshcore.Snapshot) {
	reportRow(w, "Transport", fmt.Sprintf("%s (%s)", s.Transport, s.Target))
	name := s.Name
	if name == "" {
		name = "(unnamed)"
	}
	reportRow(w, "This node", fmt.Sprintf("%s  %s", name, s.PublicKey))
	if s.Model != "" {
		reportRow(w, "Model", s.Model)
	}
	if s.Firmware != "" {
		reportRow(w, "Firmware", s.Firmware)
	}
	if r := s.Radio; r != nil {
		reportRow(w, "Radio", fmt.Sprintf("%g MHz · bw %g kHz · SF %d · CR %d · tx %d dBm",
			r.FreqMHz, r.BandwidthKHz, r.SpreadFactor, r.CodingRate, r.TxPowerDBm))
	}
	reportRow(w, "Contacts", fmt.Sprintf("%d", s.ContactCount))

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

	if len(s.Contacts) == 0 {
		fmt.Fprintln(w, "\nNo contacts in this node's list.")
		return
	}
	fmt.Fprintln(w, "\nContacts (most recently heard first):")
	fmt.Fprintf(w, "  %-20s %-12s %-6s %s\n", "Name", "Type", "Path", "Last heard")
	for _, c := range s.Contacts {
		path := "flood"
		if c.PathLen >= 0 {
			path = fmt.Sprintf("%d hop", c.PathLen)
		}
		fmt.Fprintf(w, "  %-20s %-12s %-6s %s\n",
			truncateRunes(c.Name, 20), c.Type, path, relHeard(c.LastSeen))
	}
}
