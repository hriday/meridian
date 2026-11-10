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
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hriday/meridian/internal/dnsbl"
	"github.com/hriday/meridian/internal/dnsx"
	"github.com/hriday/meridian/internal/ipspace"
	"github.com/hriday/meridian/internal/lookup"
	"github.com/hriday/meridian/internal/netcat"
	"github.com/hriday/meridian/internal/netinput"
	"github.com/hriday/meridian/internal/ntp"
	"github.com/hriday/meridian/internal/pwaudit"
	"github.com/hriday/meridian/internal/scan"
	"github.com/hriday/meridian/internal/subnet"
	"github.com/hriday/meridian/internal/sysconf"
	"github.com/hriday/meridian/internal/tlscheck"
	"github.com/hriday/meridian/internal/trace"
	"github.com/hriday/meridian/internal/tracerun"
)

// newLookupClient builds the lookup client the CLI uses; tests override it to
// inject a fake HTTP transport.
var newLookupClient = func() *lookup.Client {
	return lookup.NewClient(&http.Client{Timeout: 10 * time.Second}, 10*time.Minute)
}

// newPinger builds the ping-sweep pinger; tests override it to inject a fake.
var newPinger = func() scan.Pinger { return scan.NewICMPPinger() }

// nc seams: input piped to the connection, and the grace period to drain
// responses after stdin EOF before closing. Tests override both.
var (
	ncStdin io.Reader = os.Stdin
	ncGrace           = 200 * time.Millisecond
)

// newCracker builds the password cracker the CLI uses; tests override it.
var newCracker = func() pwaudit.Cracker { return pwaudit.NewHashcatCracker() }

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
  meridian-cli pingsweep <cidr> [--ports 22,80,443] [--timeout 1s] [--concurrency 64] [--json]
  meridian-cli portscan <host> <ports|--common> [--timeout 1s] [--concurrency 128] [--json]
  meridian-cli nc <host> <port> [--udp] [--tls] [--insecure] [--listen]
  meridian-cli bgp <prefix|ip> [--json]
  meridian-cli tls <host[:port]> [--timeout 8s] [--json]
  meridian-cli blocklist <ip> [--resolver <ip>] [--timeout 8s] [--json]
  meridian-cli adaudit --dump <file> [--wordlist <path>] [--rules <path>] [--show] [--json]
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
	case "pingsweep":
		return runPingSweep(args[1:], stdout, stderr)
	case "portscan":
		return runPortScan(args[1:], stdout, stderr)
	case "nc":
		return runNC(args[1:], stdout, stderr)
	case "bgp":
		return runBGP(args[1:], stdout, stderr)
	case "tls":
		return runTLS(args[1:], stdout, stderr)
	case "blocklist":
		return runBlocklist(args[1:], stdout, stderr)
	case "adaudit":
		return runADAudit(args[1:], stdout, stderr)
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

func runPingSweep(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pingsweep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	portsSpec := fs.String("ports", "80,443,22", "TCP ports for the fallback probe")
	timeout := fs.Duration("timeout", time.Second, "per-host timeout")
	concurrency := fs.Int("concurrency", 64, "max concurrent probes")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "pingsweep needs a CIDR\n%s", usage)
		return 2
	}
	tcpPorts, err := scan.ParsePorts(*portsSpec)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	res, err := scan.Sweep(context.Background(), pos[0], newPinger(), tcpPorts, *timeout, *concurrency, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	up := 0
	for _, h := range res {
		if h.Alive {
			up++
			name := h.Name
			if name != "" {
				name = "  " + name
			}
			fmt.Fprintf(stdout, "%-15s up (%s)%s\n", h.IP, h.Method, name)
		}
	}
	fmt.Fprintf(stdout, "%d of %d hosts up\n", up, len(res))
	return 0
}

func runPortScan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("portscan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	common := fs.Bool("common", false, "scan a preset of common ports")
	timeout := fs.Duration("timeout", time.Second, "per-port timeout")
	concurrency := fs.Int("concurrency", 128, "max concurrent connects")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) < 1 {
		fmt.Fprintf(stderr, "portscan needs a host\n%s", usage)
		return 2
	}
	var ports []int
	var err error
	if *common {
		ports = scan.CommonPorts()
	} else {
		if len(pos) < 2 {
			fmt.Fprintf(stderr, "portscan needs ports (or --common)\n%s", usage)
			return 2
		}
		if ports, err = scan.ParsePorts(pos[1]); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	res, err := scan.ScanPorts(context.Background(), pos[0], ports, *timeout, *concurrency, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	open := 0
	for _, r := range res {
		if r.State == "open" {
			open++
			fmt.Fprintf(stdout, "%5d/tcp  open  %s\n", r.Port, r.Service)
		}
	}
	fmt.Fprintf(stdout, "%d open of %d scanned\n", open, len(res))
	return 0
}

type ncWriter struct{ s *netcat.Session }

func (w ncWriter) Write(b []byte) (int, error) { return w.s.Send(b) }

func runNC(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("nc", flag.ContinueOnError)
	fs.SetOutput(stderr)
	udp := fs.Bool("udp", false, "use UDP")
	useTLS := fs.Bool("tls", false, "wrap the connection in TLS")
	insecure := fs.Bool("insecure", false, "skip TLS verification")
	listen := fs.Bool("listen", false, "listen on the local port instead of connecting")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 2 {
		fmt.Fprintf(stderr, "nc needs <host> <port>\n%s", usage)
		return 2
	}
	network := "tcp"
	if *udp {
		network = "udp"
	}
	cfg := netcat.Config{Network: network, Addr: net.JoinHostPort(pos[0], pos[1]), TLS: *useTLS, Insecure: *insecure}
	ctx := context.Background()

	var s *netcat.Session
	if *listen {
		l, err := netcat.Listen(ctx, cfg)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer l.Close()
		fmt.Fprintf(stderr, "listening on %s (%s)\n", l.Addr(), network)
		if s, err = l.Accept(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		var cert *netcat.CertInfo
		var err error
		if s, cert, err = netcat.Dial(ctx, cfg); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if cert != nil {
			fmt.Fprintf(stderr, "TLS subject %s · expires %s · verified=%v\n", cert.Subject, cert.NotAfter, cert.Verified)
		}
	}
	defer s.Close()

	var mu sync.Mutex
	remoteClosed := make(chan struct{})
	s.Start(func(b []byte) {
		mu.Lock()
		_, _ = stdout.Write(b)
		mu.Unlock()
	}, func(error) { close(remoteClosed) })

	in := ncStdin // snapshot the seam so a late goroutine read can't race a test reset
	stdinDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(ncWriter{s}, in)
		close(stdinDone)
	}()

	// Exit when the remote hangs up, or shortly after our own input ends.
	select {
	case <-remoteClosed:
	case <-stdinDone:
		select {
		case <-remoteClosed:
		case <-time.After(ncGrace): // drain late responses, then close
		}
	}
	return 0
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
	reportRow(stdout, "Peering", peeringLine(r.Peering))
	reportRow(stdout, "Registered", strings.TrimSpace(r.RegisteredOrg+registrySuffix(r.Registry)))
	reportRow(stdout, "Abuse", r.AbuseEmail)
	reportRow(stdout, "Note", r.RoutingNote)
	printSourceErrs(stdout, r.RoutingSrc, r.RegistrySrc, r.PeeringSrc)
}

// peeringLine is a one-line PeeringDB summary of the origin AS for the IP page:
// "<n> IXPs · <m> facilities · policy <policy>". Counts come from the fetched
// lists; an empty section yields "" so the row is suppressed.
func peeringLine(p *lookup.Peering) string {
	if p == nil {
		return ""
	}
	var parts []string
	if n := len(p.IXPs); n > 0 {
		parts = append(parts, peeringPresence(n, "IXP", "IXPs", nil))
	}
	if n := len(p.Facilities); n > 0 {
		parts = append(parts, peeringPresence(n, "facility", "facilities", nil))
	}
	if p.Policy != "" {
		parts = append(parts, "policy "+p.Policy)
	}
	return strings.Join(parts, " · ")
}

func printASNReport(stdout io.Writer, r lookup.ASNReport) {
	reportRow(stdout, "ASN", r.ASN)
	reportRow(stdout, "Holder", strings.TrimSpace(r.Holder+registrySuffix(r.Registry)))
	reportRow(stdout, "Prefixes", fmt.Sprintf("%d IPv4, %d IPv6", r.V4Prefixes, r.V6Prefixes))
	reportRow(stdout, "Neighbours", fmt.Sprintf("%d", r.Neighbours))
	if p := r.Peering; p != nil {
		summary := p.Type
		if p.Policy != "" {
			summary = strings.TrimSpace(summary + " · policy " + p.Policy)
		}
		reportRow(stdout, "Peering", summary)
		reportRow(stdout, "Website", p.Website)
		reportRow(stdout, "IXPs", peeringPresence(len(p.IXPs), "IXP", "IXPs", topIXNames(p.IXPs)))
		reportRow(stdout, "Facilities", peeringPresence(len(p.Facilities), "facility", "facilities", topFacNames(p.Facilities)))
	}
	printSourceErrs(stdout, r.RoutingSrc, r.RegistrySrc, r.PeeringSrc)
}

// peeringPresence renders "<count> <noun> — <sample>, +N more" for a PeeringDB
// presence, choosing the singular/plural noun by count. The count is the length
// of the fetched (deduped) list, so it and the "+N more" never disagree; an
// empty presence returns "" so the row is suppressed.
func peeringPresence(count int, singular, plural string, sample []string) string {
	if count == 0 {
		return ""
	}
	noun := plural
	if count == 1 {
		noun = singular
	}
	s := fmt.Sprintf("%d %s", count, noun)
	if len(sample) > 0 {
		s += " — " + strings.Join(sample, ", ")
		if count > len(sample) {
			s += fmt.Sprintf(", +%d more", count-len(sample))
		}
	}
	return s
}

func topIXNames(ixps []lookup.PeerIX) []string {
	n := len(ixps)
	if n > 3 {
		n = 3
	}
	out := make([]string, 0, n)
	for _, ix := range ixps[:n] {
		out = append(out, ix.Name)
	}
	return out
}

func topFacNames(facs []lookup.PeerFac) []string {
	n := len(facs)
	if n > 3 {
		n = 3
	}
	out := make([]string, 0, n)
	for _, f := range facs[:n] {
		out = append(out, f.Name)
	}
	return out
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

func runTLS(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tls", flag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 8*time.Second, "connection timeout")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "tls needs exactly one host\n%s", usage)
		return 2
	}
	rep, err := tlscheck.Check(context.Background(), pos[0], *timeout)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printTLSReport(stdout, rep)
	return 0
}

func printTLSReport(stdout io.Writer, r tlscheck.CertReport) {
	reportRow(stdout, "Verdict", r.Verdict)
	reportRow(stdout, "Host", strings.TrimSpace(r.Host+":"+r.Port))
	reportRow(stdout, "Subject", r.Subject)
	reportRow(stdout, "Issuer", r.Issuer)
	if len(r.SANs) > 0 {
		reportRow(stdout, "SANs", strings.Join(r.SANs, ", "))
	}
	if !r.NotAfter.IsZero() {
		reportRow(stdout, "Valid until", fmt.Sprintf("%s (%d days)", r.NotAfter.Format("2006-01-02"), r.DaysToExpiry))
	}
	if r.KeyType != "" {
		reportRow(stdout, "Key", fmt.Sprintf("%s %d-bit", r.KeyType, r.KeyBits))
	}
	reportRow(stdout, "Signature", r.SigAlg)
	reportRow(stdout, "TLS", strings.TrimSpace(r.TLSVersion+"  "+r.CipherSuite))
	if len(r.Warnings) > 0 {
		reportRow(stdout, "Warnings", strings.Join(r.Warnings, "; "))
	}
}

func runBlocklist(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("blocklist", flag.ContinueOnError)
	fs.SetOutput(stderr)
	resolver := fs.String("resolver", "", "resolver to query (default: system)")
	timeout := fs.Duration("timeout", 8*time.Second, "overall timeout")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "blocklist needs exactly one IP\n%s", usage)
		return 2
	}
	srv := *resolver
	if srv == "" {
		if f := sysconf.Resolvers(); len(f.Values) > 0 {
			srv = f.Values[0]
		} else {
			srv = "1.1.1.1"
		}
	}
	rep, err := dnsbl.Check(context.Background(), pos[0], srv, dnsbl.DefaultZones(), *timeout)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printBlocklistReport(stdout, rep)
	return 0
}

func printBlocklistReport(stdout io.Writer, r dnsbl.Report) {
	reportRow(stdout, "Verdict", r.Verdict)
	reportRow(stdout, "IP", r.IP)
	reportRow(stdout, "Resolver", r.Resolver)
	reportRow(stdout, "Summary", r.Summary)
	fmt.Fprintln(stdout)
	for _, l := range r.Listings {
		line := fmt.Sprintf("%-10s %s (%s)", string(l.Status), l.Name, l.Zone)
		if len(l.Codes) > 0 {
			line += "  " + strings.Join(l.Codes, ",")
		}
		fmt.Fprintln(stdout, line)
		if l.Reason != "" {
			fmt.Fprintf(stdout, "           reason: %s\n", l.Reason)
		}
		if l.Note != "" {
			fmt.Fprintf(stdout, "           note: %s\n", l.Note)
		}
	}
}

func runADAudit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("adaudit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dump := fs.String("dump", "", "secretsdump/pwdump file to audit")
	wl := fs.String("wordlist", "", "wordlist file for the crack")
	rulesFile := fs.String("rules", "", "optional hashcat rules file")
	show := fs.Bool("show", false, "reveal recovered plaintext passwords")
	asJSON := fs.Bool("json", false, "output JSON")
	if _, ok := parsePositionals(fs, args); !ok {
		return 2
	}
	if *dump == "" {
		fmt.Fprintf(stderr, "adaudit needs --dump <file>\n%s", usage)
		return 2
	}
	f, err := os.Open(*dump)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer f.Close()
	accounts, err := pwaudit.ParseDump(f)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	var rules []string
	if *rulesFile != "" {
		rules = []string{*rulesFile}
	}
	rep, err := pwaudit.RunAudit(context.Background(), accounts, newCracker(), *wl, rules, nil)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		if !*show {
			rep = redactReport(rep)
		}
		return writeJSON(stdout, stderr, rep)
	}
	printADAuditReport(stdout, rep, *show)
	return 0
}

// redactReport blanks recovered plaintext for output when --show is not set.
func redactReport(r pwaudit.AuditReport) pwaudit.AuditReport {
	for i := range r.Results {
		if r.Results[i].Cracked {
			r.Results[i].Plaintext = "••••••"
		}
	}
	for i := range r.Reuse {
		if r.Reuse[i].Plaintext != "" && r.Reuse[i].Plaintext != "(empty password)" {
			r.Reuse[i].Plaintext = "••••••"
		}
	}
	return r
}

func printADAuditReport(stdout io.Writer, r pwaudit.AuditReport, show bool) {
	reportRow(stdout, "Cracker", r.CrackerUsed)
	reportRow(stdout, "Accounts", fmt.Sprintf("%d", r.Total))
	reportRow(stdout, "Cracked", fmt.Sprintf("%d (%.0f%%)", r.Cracked, r.PercentCracked))
	for _, p := range r.Patterns {
		reportRow(stdout, "Pattern", fmt.Sprintf("%s × %d", p.Pattern, p.Count))
	}
	if len(r.Reuse) > 0 {
		reportRow(stdout, "Reused", fmt.Sprintf("%d group(s)", len(r.Reuse)))
	}
	for _, res := range r.Results {
		if !res.Cracked {
			continue
		}
		pw := "••••••"
		if show {
			pw = res.Plaintext
		}
		fmt.Fprintf(stdout, "%-20s %-8s %-14s %s\n", res.Account.Name, res.Strength, res.Pattern, pw)
	}
	if !show {
		fmt.Fprintln(stdout, "(passwords hidden; re-run with --show to reveal)")
	}
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
