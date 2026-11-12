package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/snmp"
)

// A small default community list for the audit command, used when the operator
// doesn't supply one. These are the conventional defaults an audit should flag.
var defaultCommunities = []string{
	"public", "private", "community", "cisco", "admin", "read", "write",
	"snmp", "manager", "default", "secret", "test", "router", "switch", "monitor",
}

// scrubCommunities removes any of the given community strings from s, so a
// community never reaches stdout/stderr via an error message.
func scrubCommunities(s string, comms []string) string {
	for _, c := range comms {
		if c != "" {
			s = strings.ReplaceAll(s, c, "***")
		}
	}
	return s
}

// ---- snmpsweep -------------------------------------------------------------

func runSNMPSweep(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("snmpsweep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	community := fs.String("community", "public", "SNMP v2c read community")
	timeout := fs.Duration("timeout", 2*time.Second, "per-host timeout")
	conc := fs.Int("concurrency", 64, "parallel probes")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "snmpsweep needs exactly one CIDR\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	hosts, err := snmp.Sweep(ctx, pos[0], *community, *timeout, *conc, nil)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, hosts)
	}
	fmt.Fprintf(stdout, "%d SNMP responder(s)\n", len(hosts))
	for _, h := range hosts {
		fmt.Fprintf(stdout, "%-16s %-24s %s\n", h.IP, h.Name, firstLine(h.Descr, 70))
	}
	return 0
}

// ---- devhealth -------------------------------------------------------------

type devHealthOut struct {
	Device     string  `json:"device"`
	Name       string  `json:"name"`
	Descr      string  `json:"descr"`
	UptimeSecs uint32  `json:"uptimeSecs"`
	CPUPct     float64 `json:"cpuPct"`
	MemPct     float64 `json:"memPct"`
	HasCPU     bool    `json:"hasCpu"`
	HasMem     bool    `json:"hasMem"`
	Interfaces int     `json:"interfaces"`
	IfUp       int     `json:"ifUp"`
}

func runDevHealth(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("devhealth", flag.ContinueOnError)
	fs.SetOutput(stderr)
	community := fs.String("community", "public", "SNMP v2c read community")
	timeout := fs.Duration("timeout", 2*time.Second, "per-request timeout")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "devhealth needs exactly one device\n%s", usage)
		return 2
	}
	c, err := snmp.Dial(pos[0], *community, *timeout)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	defer c.Close()
	sys, err := snmp.System(c)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	res, _ := snmp.ResourceUsage(c) // best-effort; never errors
	ifs, _ := snmp.ListInterfaces(c)
	up := 0
	for _, i := range ifs {
		if i.OperStatus == "up" {
			up++
		}
	}
	out := devHealthOut{
		Device: pos[0], Name: sys.Name, Descr: sys.Descr, UptimeSecs: sys.UptimeTicks / 100,
		CPUPct: res.CPUPct, MemPct: res.MemPct, HasCPU: res.HasCPU, HasMem: res.HasMem,
		Interfaces: len(ifs), IfUp: up,
	}
	if *asJSON {
		return writeJSON(stdout, stderr, out)
	}
	printDevHealth(stdout, out)
	return 0
}

func printDevHealth(stdout io.Writer, o devHealthOut) {
	reportRow(stdout, "Device", o.Device)
	reportRow(stdout, "Name", o.Name)
	reportRow(stdout, "Description", firstLine(o.Descr, 90))
	reportRow(stdout, "Uptime", (time.Duration(o.UptimeSecs) * time.Second).String())
	reportRow(stdout, "CPU", pctOrNA(o.CPUPct, o.HasCPU))
	reportRow(stdout, "Memory", pctOrNA(o.MemPct, o.HasMem))
	reportRow(stdout, "Interfaces", fmt.Sprintf("%d (%d up)", o.Interfaces, o.IfUp))
}

func pctOrNA(v float64, has bool) string {
	if !has {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", v)
}

// ---- swport ----------------------------------------------------------------

func runSwitchPort(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("swport", flag.ContinueOnError)
	fs.SetOutput(stderr)
	community := fs.String("community", "public", "SNMP v2c read community")
	timeout := fs.Duration("timeout", 2*time.Second, "per-request timeout")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "swport needs exactly one switch device\n%s", usage)
		return 2
	}
	c, err := snmp.Dial(pos[0], *community, *timeout)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	defer c.Close()
	ports, err := snmp.SwitchPorts(c)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, ports)
	}
	fmt.Fprintf(stdout, "%d MAC entries\n", len(ports))
	for _, p := range ports {
		vlan := "—"
		if p.VLAN != 0 {
			vlan = fmt.Sprintf("%d", p.VLAN)
		}
		fmt.Fprintf(stdout, "%-18s %-20s vlan %-5s %s\n", p.MAC, p.Port, vlan, p.IP)
	}
	return 0
}

// ---- mib -------------------------------------------------------------------

func runMIB(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mib", flag.ContinueOnError)
	fs.SetOutput(stderr)
	community := fs.String("community", "public", "SNMP v2c read community")
	timeout := fs.Duration("timeout", 2*time.Second, "per-request timeout")
	root := fs.String("root", "1.3.6.1.2.1.1", "root OID to walk (default: system)")
	max := fs.Int("max", 500, "max rows")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "mib needs exactly one device\n%s", usage)
		return 2
	}
	c, err := snmp.Dial(pos[0], *community, *timeout)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	defer c.Close()
	rows, err := snmp.MIBWalk(c, *root, *max)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), []string{*community}))
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rows)
	}
	for _, r := range rows {
		fmt.Fprintf(stdout, "%-30s %-12s %s\n", r.OID, r.Type, firstLine(r.Value, 80))
	}
	return 0
}

// ---- snmpbrute -------------------------------------------------------------

func runSNMPBrute(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("snmpbrute", flag.ContinueOnError)
	fs.SetOutput(stderr)
	list := fs.String("communities", "", "comma-separated community strings (default: a built-in list)")
	file := fs.String("list", "", "file of community strings, one per line")
	timeout := fs.Duration("timeout", time.Second, "per-try timeout")
	conc := fs.Int("concurrency", 10, "parallel tries")
	authorized := fs.Bool("authorized", false, "confirm you are authorized to audit this device")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "snmpbrute needs exactly one device\n%s", usage)
		return 2
	}
	if !*authorized {
		fmt.Fprintln(stderr, "refusing: pass --authorized to confirm you are allowed to audit this device")
		return 2
	}

	comms, err := communityList(*list, *file)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	found, err := snmp.BruteCommunities(ctx, pos[0], comms, *timeout, *conc)
	if err != nil {
		fmt.Fprintln(stderr, scrubCommunities(err.Error(), comms))
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, found)
	}
	if len(found) == 0 {
		fmt.Fprintf(stdout, "no working community found among %d tried\n", len(comms))
		return 0
	}
	fmt.Fprintf(stdout, "%d working community string(s) — rotate these:\n", len(found))
	for _, c := range found {
		fmt.Fprintf(stdout, "  %s\n", c)
	}
	return 0
}

// communityList resolves the community strings to try: an explicit --list file,
// else a --communities CSV, else the built-in default set.
func communityList(csv, file string) ([]string, error) {
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		var out []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if s := strings.TrimSpace(sc.Text()); s != "" {
				out = append(out, s)
			}
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("community list %q is empty", file)
		}
		return out, nil
	}
	if csv != "" {
		var out []string
		for _, s := range strings.Split(csv, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	}
	return defaultCommunities, nil
}
