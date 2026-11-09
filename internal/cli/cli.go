// Package cli is the command-line face of Meridian's engines: the same
// Classify and subnet logic the GUI uses, scriptable with --json. Run is
// the testable entry point; cmd/meridian-cli is a thin wrapper over it.
package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/hriday/engineers_tools/internal/netinput"
	"github.com/hriday/engineers_tools/internal/subnet"
)

const usage = `meridian-cli — network info toolkit

usage:
  meridian-cli classify <value> [--json]
  meridian-cli subnet <cidr> [--split <prefixlen>] [--json]
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
