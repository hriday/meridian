// Package cli is the command-line face of Meridian's engines: the same
// Classify and subnet logic the GUI uses, scriptable with --json. Run is
// the testable entry point; cmd/meridian-cli is a thin wrapper over it.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/ipspace"
	"github.com/hriday/meridian/internal/lookup"
	"github.com/hriday/meridian/internal/netinput"
	"github.com/hriday/meridian/internal/ntp"
	"github.com/hriday/meridian/internal/subnet"
	"github.com/hriday/meridian/internal/sysconf"
	"github.com/hriday/meridian/internal/trace"
	"github.com/hriday/meridian/internal/tracerun"
)

// newLookupClient builds the lookup client the CLI uses; tests override it to
// inject a fake HTTP transport.
var newLookupClient = func() *lookup.Client {
	return lookup.NewClient(&http.Client{Timeout: 10 * time.Second}, 10*time.Minute)
}

// Query functions behind package vars so tests inject fakes.
var (
	ntpQuery = ntp.Query
	dnsQuery = dnsx.Query
)

const usage = `meridian-cli — network info toolkit

usage:
  meridian-cli classify <value> [--json]
  meridian-cli subnet <cidr> [--split <prefixlen>] [--json]
  meridian-cli lookup <ip|asn> [--json]
  meridian-cli ntp <server> [--json]
  meridian-cli dns <name> [--type A] [--server <ip>] [--json]
  meridian-cli trace <host> [--6] [--json]
  meridian-cli bgp <prefix|ip> [--json]
  meridian-cli ipspace list|export [--file <path>] [--passphrase <pw>]
`

// Run dispatches args (without the program name), writing to stdout/stderr,
// and returns a process exit code (0 ok, 2 usage error, 1 runtime error).
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "classify":
		return runClassify(args[1:], stdout, stderr)
	case "subnet":
		return runSubnet(args[1:], stdout, stderr)
	case "lookup":
		return runLookup(args[1:], stdout, stderr)
	case "ntp":
		return runNTP(args[1:], stdout, stderr)
	case "dns":
		return runDNS(args[1:], stdout, stderr)
	case "trace":
		return runTrace(args[1:], stdout, stderr)
	case "bgp":
		return runBGP(args[1:], stdout, stderr)
	case "ipspace":
		return runIPSpace(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n%s", args[0], usage)
		return 2
	}
}

// parsePositionals parses flags that may appear before, after, or around
// the positional arguments (Go's flag stops at the first non-flag, so we
// resume after taking each positional). Returns the positionals, or nil and
// false on a flag parse error.
func parsePositionals(fs *flag.FlagSet, args []string) ([]string, bool) {
	var pos []string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return nil, false
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
	return pos, true
}

func runClassify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("classify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "classify needs exactly one value\n%s", usage)
		return 2
	}
	r, err := netinput.Classify(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	fmt.Fprintf(stdout, "%s\t%s\n", r.Kind, r.Normalized)
	return 0
}

func runSubnet(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("subnet", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	split := fs.Int("split", 0, "split into subnets of this prefix length")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "subnet needs exactly one CIDR\n%s", usage)
		return 2
	}
	cidr := pos[0]

	if *split > 0 {
		subs, err := subnet.Split(cidr, *split)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, subs)
		}
		for _, s := range subs {
			fmt.Fprintln(stdout, s)
		}
		return 0
	}

	info, err := subnet.Describe(cidr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, info)
	}
	printInfo(stdout, info)
	return 0
}

func runLookup(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lookup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "lookup needs exactly one value\n%s", usage)
		return 2
	}
	r, err := netinput.Classify(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	client := newLookupClient()
	ctx := context.Background()
	switch r.Kind {
	case netinput.KindIPv4, netinput.KindIPv6:
		rep, err := client.LookupIP(ctx, r.Normalized)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, rep)
		}
		printIPReport(stdout, rep)
		return 0
	case netinput.KindASN:
		rep, err := client.LookupASN(ctx, r.Normalized)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, rep)
		}
		printASNReport(stdout, rep)
		return 0
	case netinput.KindPrefixV4, netinput.KindPrefixV6:
		rep, err := client.LookupPrefix(ctx, r.Normalized)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, rep)
		}
		printPrefixReport(stdout, rep)
		return 0
	case netinput.KindDomain, netinput.KindHostname:
		rep, err := client.LookupDomain(ctx, r.Normalized)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, rep)
		}
		printDomainReport(stdout, rep)
		return 0
	default:
		fmt.Fprintf(stderr, "lookup needs an IP, ASN, prefix, or domain; %q is a %s\n", pos[0], r.Kind)
		return 2
	}
}

func runNTP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ntp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "ntp needs exactly one server\n%s", usage)
		return 2
	}
	rep, err := ntpQuery(context.Background(), pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	reportRow(stdout, "Server", rep.Server)
	reportRow(stdout, "Stratum", fmt.Sprintf("%d", rep.Stratum))
	reportRow(stdout, "Reference", rep.RefID)
	reportRow(stdout, "Leap", rep.LeapIndicator)
	reportRow(stdout, "Offset", rep.Offset.String())
	reportRow(stdout, "Round-trip", rep.RTT.String())
	reportRow(stdout, "Root delay", rep.RootDelay.String())
	reportRow(stdout, "Root disp", rep.RootDispersion.String())
	reportRow(stdout, "Server time", rep.ServerTime.Format("2006-01-02 15:04:05 MST"))
	if rep.KissOfDeath {
		reportRow(stdout, "Note", "kiss-o'-death (rate-limited / unusable)")
	}
	return 0
}

func runDNS(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dns", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	rtype := fs.String("type", "A", "record type")
	server := fs.String("server", "", "resolver to query (default: system)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "dns needs exactly one name\n%s", usage)
		return 2
	}
	srv := *server
	if srv == "" {
		if f := sysconf.Resolvers(); len(f.Values) > 0 {
			srv = f.Values[0]
		} else {
			srv = "1.1.1.1"
		}
	}
	res, err := dnsQuery(context.Background(), pos[0], *rtype, srv)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	reportRow(stdout, "Server", res.Server)
	reportRow(stdout, "Status", res.Rcode)
	reportRow(stdout, "Flags", strings.Join(res.Flags, " "))
	for _, a := range res.Answers {
		fmt.Fprintf(stdout, "%-14s %s\t%d\t%s\n", a.Type, a.Name, a.TTL, a.Value)
	}
	return 0
}

func runTrace(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("trace", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	useV6 := fs.Bool("6", false, "trace over IPv6")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "trace needs exactly one host\n%s", usage)
		return 2
	}
	r, err := tracerun.Run(context.Background(), pos[0], *useV6, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, r)
	}
	for _, h := range r.Hops {
		host := h.Addr
		if h.Timeout {
			host = "*"
		} else if h.Host != "" {
			host = fmt.Sprintf("%s (%s)", h.Host, h.Addr)
		}
		as := "—"
		if h.ASN != "" {
			as = "AS" + h.ASN
			if h.ASOwner != "" {
				as += " " + h.ASOwner
			}
		}
		_, avg, _ := traceRTT(h.RTTs)
		rtt := "*"
		if avg > 0 {
			rtt = fmt.Sprintf("%.1f ms", float64(avg)/1e6)
		}
		fmt.Fprintf(stdout, "%2d  %-40s %-26s %s\n", h.TTL, host, as, rtt)
	}
	fmt.Fprintln(stdout, trace.Verdict(r))
	return 0
}

func traceRTT(d []time.Duration) (min, avg, max time.Duration) {
	var sum time.Duration
	var n int
	for _, x := range d {
		if x <= 0 {
			continue
		}
		if n == 0 || x < min {
			min = x
		}
		if x > max {
			max = x
		}
		sum += x
		n++
	}
	if n == 0 {
		return 0, 0, 0
	}
	return min, sum / time.Duration(n), max
}

func runIPSpace(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "ipspace needs a subcommand (list or export)\n%s", usage)
		return 2
	}
	sub := args[0]
	fs := flag.NewFlagSet("ipspace", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", ipspace.DefaultPath(), "store file path")
	pass := fs.String("passphrase", "", "store passphrase (or MERIDIAN_PASSPHRASE)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	passphrase := *pass
	if passphrase == "" {
		passphrase = os.Getenv("MERIDIAN_PASSPHRASE")
	}
	events, err := ipspace.Load(*file, passphrase)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	store := ipspace.FromEvents(ipspace.Actor(), events)
	switch sub {
	case "list":
		for _, r := range store.Records() {
			name := r.Name
			if name == "" {
				name = "—"
			}
			fmt.Fprintf(stdout, "%-20s %-10s %s\n", r.Prefix, r.Status, name)
		}
		return 0
	case "export":
		if err := store.ExportCSV(stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "unknown ipspace subcommand %q (want list or export)\n", sub)
		return 2
	}
}

func writeJSON(stdout, stderr io.Writer, v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(b))
	return 0
}

func printInfo(stdout io.Writer, i subnet.Info) {
	row := func(k, v string) {
		if v != "" {
			fmt.Fprintf(stdout, "%-12s %s\n", k, v)
		}
	}
	row("Prefix", i.Prefix)
	row("Network", i.Network)
	row("Broadcast", i.Broadcast)
	row("Netmask", i.Netmask)
	row("Wildcard", i.Wildcard)
	row("Addresses", i.TotalCount)
	row("Usable", fmt.Sprintf("%s – %s (%s)", i.UsableFirst, i.UsableLast, i.UsableCount))
}

func reportRow(stdout io.Writer, k, v string) {
	if v != "" {
		fmt.Fprintf(stdout, "%-14s %s\n", k, v)
	}
}

func printIPReport(stdout io.Writer, r lookup.IPReport) {
	reportRow(stdout, "Address", fmt.Sprintf("%s (%s)", r.Input, r.Version))
	reportRow(stdout, "Prefix", r.CoveringPrefix)
	if r.OriginASN != "" {
		reportRow(stdout, "Origin AS", strings.TrimSpace(r.OriginASN+"  "+r.OriginName))
	}
	reportRow(stdout, "Registered", strings.TrimSpace(r.RegisteredOrg+registrySuffix(r.Registry)))
	reportRow(stdout, "Abuse", r.AbuseEmail)
	reportRow(stdout, "Note", r.RoutingNote)
	printSourceErrs(stdout, r.RoutingSrc, r.RegistrySrc)
}

func printASNReport(stdout io.Writer, r lookup.ASNReport) {
	reportRow(stdout, "ASN", r.ASN)
	reportRow(stdout, "Holder", strings.TrimSpace(r.Holder+registrySuffix(r.Registry)))
	reportRow(stdout, "Prefixes", fmt.Sprintf("%d IPv4, %d IPv6", r.V4Prefixes, r.V6Prefixes))
	reportRow(stdout, "Neighbours", fmt.Sprintf("%d", r.Neighbours))
	printSourceErrs(stdout, r.RoutingSrc, r.RegistrySrc)
}

func printPrefixReport(stdout io.Writer, r lookup.PrefixReport) {
	reportRow(stdout, "Prefix", r.Input)
	if r.OriginASN != "" {
		reportRow(stdout, "Origin AS", strings.TrimSpace(r.OriginASN+"  "+r.OriginName))
	}
	reportRow(stdout, "Announced", fmt.Sprintf("%t", r.Announced))
	reportRow(stdout, "RPKI", strings.TrimSpace(r.RPKIState+"  "+r.RPKIRoa))
	reportRow(stdout, "Registered", strings.TrimSpace(r.RegisteredOrg+registrySuffix(r.Registry)))
	reportRow(stdout, "Abuse", r.AbuseEmail)
	if len(r.LessSpecific) > 0 {
		reportRow(stdout, "Less specific", strings.Join(r.LessSpecific, ", "))
	}
	if len(r.MoreSpecific) > 0 {
		reportRow(stdout, "More specific", strings.Join(r.MoreSpecific, ", "))
	}
	printSourceErrs(stdout, r.RoutingSrc, r.RPKISrc, r.RegistrySrc)
}

func printDomainReport(stdout io.Writer, r lookup.DomainReport) {
	reportRow(stdout, "Domain", r.Input)
	if len(r.ResolvesTo) > 0 {
		reportRow(stdout, "Resolves to", strings.Join(r.ResolvesTo, ", "))
	}
	if len(r.HostedBy) > 0 {
		reportRow(stdout, "Hosted by", strings.Join(r.HostedBy, ", "))
	}
	reportRow(stdout, "Registrar", r.Registrar)
	reportRow(stdout, "Created", r.Created)
	reportRow(stdout, "Expires", r.Expires)
	reportRow(stdout, "DNSSEC", r.DNSSEC)
	if len(r.Nameservers) > 0 {
		reportRow(stdout, "Nameservers", strings.Join(r.Nameservers, ", "))
	}
	for _, rec := range r.Records {
		fmt.Fprintf(stdout, "%-14s %s\t%s\n", rec.Type, rec.Name, rec.Value)
	}
	printSourceErrs(stdout, r.DNSSrc, r.RegistrySrc)
}

func runBGP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("bgp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "bgp needs exactly one prefix or IP\n%s", usage)
		return 2
	}
	r, err := netinput.Classify(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	switch r.Kind {
	case netinput.KindIPv4, netinput.KindIPv6, netinput.KindPrefixV4, netinput.KindPrefixV6:
	default:
		fmt.Fprintf(stderr, "bgp needs a prefix or IP; %q is a %s\n", pos[0], r.Kind)
		return 2
	}
	rep, err := newLookupClient().LookupBGP(context.Background(), r.Normalized)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printBGPReport(stdout, rep)
	return 0
}

func printBGPReport(stdout io.Writer, r lookup.BGPReport) {
	reportRow(stdout, "Prefix", r.Prefix)
	if len(r.OriginASNs) > 0 {
		origin := "AS" + strings.Join(r.OriginASNs, ", AS")
		if r.MOAS {
			origin += "  (MOAS: multiple origins)"
		}
		reportRow(stdout, "Origin AS", origin)
	}
	reportRow(stdout, "Visibility", fmt.Sprintf("seen by %d peers across %d collectors · %d distinct paths",
		r.Peers, r.Collectors, len(r.Paths)))
	if r.RPKIState != "" {
		reportRow(stdout, "RPKI", r.RPKIState)
	}
	for i, p := range r.Paths {
		if i >= 5 {
			break
		}
		path := "AS" + strings.Join(p.Path, " → AS")
		reportRow(stdout, fmt.Sprintf("Path %d", i+1),
			fmt.Sprintf("%s  (%d peers, %d%%)", path, p.Peers, p.Percent))
	}
	if len(r.Upstreams) > 0 {
		reportRow(stdout, "Upstreams", "AS"+strings.Join(r.Upstreams, ", AS"))
	}
	if len(r.Transit) > 0 {
		reportRow(stdout, "Transit", "AS"+strings.Join(r.Transit, ", AS"))
	}
	printSourceErrs(stdout, r.RoutingSrc, r.RPKISrc)
}

func registrySuffix(registry string) string {
	if registry == "" {
		return ""
	}
	return "  (" + registry + ")"
}

func printSourceErrs(stdout io.Writer, sources ...lookup.Source) {
	for _, s := range sources {
		if s.Err != "" {
			fmt.Fprintf(stdout, "%-14s %s: %s\n", "(unavailable)", s.Label, s.Err)
		}
	}
}
