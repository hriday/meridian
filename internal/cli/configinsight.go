package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/ciscopw"
	"github.com/hriday/meridian/internal/configdiff"
	"github.com/hriday/meridian/internal/hibp"
	"github.com/hriday/meridian/internal/tfscan"
	"github.com/hriday/meridian/internal/xon"
)

// Stdin seams for the file-or-stdin commands; tests override them.
var (
	ciscopwStdin io.Reader = os.Stdin
	tfStdin      io.Reader = os.Stdin
)

// ---- ciscopw ---------------------------------------------------------------

func runCiscoPW(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ciscopw", flag.ContinueOnError)
	fs.SetOutput(stderr)
	decode := fs.String("decode", "", "decode a single Cisco type 7 value and exit")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}

	if *decode != "" {
		pt, err := ciscopw.DecodeType7(*decode)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, map[string]string{"encoded": *decode, "plaintext": pt})
		}
		fmt.Fprintln(stdout, pt)
		return 0
	}

	data, err := readFileOrStdin(pos, ciscopwStdin)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	secrets := ciscopw.Analyze(string(data))
	if *asJSON {
		return writeJSON(stdout, stderr, secrets)
	}
	printCiscoSecrets(stdout, secrets)
	return 0
}

func printCiscoSecrets(stdout io.Writer, secrets []ciscopw.Secret) {
	if len(secrets) == 0 {
		fmt.Fprintln(stdout, "no password/secret commands found")
		return
	}
	for _, s := range secrets {
		line := fmt.Sprintf("%-10s type %-2s %-20s %s", s.Verdict, s.Type, s.Keyword, s.Context)
		fmt.Fprintln(stdout, line)
		if s.Decoded && s.Plaintext != "" {
			fmt.Fprintf(stdout, "           recovered: %s\n", s.Plaintext)
		} else if s.Encoded != "" {
			fmt.Fprintf(stdout, "           stored: %s\n", s.Encoded)
		}
	}
}

// ---- tfexposure ------------------------------------------------------------

func runTFExposure(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tfexposure", flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("source", "", "input kind: hcl | plan (default: auto-detect)")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	data, err := readFileOrStdin(pos, tfStdin)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	an, err := tfscan.Analyze(string(data), *source)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, an)
	}
	printTFScan(stdout, an)
	return 0
}

func printTFScan(stdout io.Writer, a tfscan.Analysis) {
	c := a.Counts
	fmt.Fprintf(stdout, "source %s — %d resources, %d security groups, %d exposed, %d critical, %d findings\n\n",
		a.Source, c.Resources, c.SecurityGroups, c.Exposed, c.Critical, c.Findings)
	for _, f := range a.Findings {
		loc := f.Resource
		if f.File != "" {
			loc = fmt.Sprintf("%s (%s:%d)", f.Resource, f.File, f.Line)
		}
		fmt.Fprintf(stdout, "[%-8s] %s — %s\n", strings.ToUpper(f.Severity), f.Title, loc)
		if f.Detail != "" {
			fmt.Fprintf(stdout, "           %s\n", f.Detail)
		}
	}
	for _, n := range a.Notes {
		fmt.Fprintf(stdout, "note: %s\n", n)
	}
}

// ---- cfgdiff ---------------------------------------------------------------

func runCfgDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cfgdiff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 2 {
		fmt.Fprintf(stderr, "cfgdiff needs two files: <old> <new>\n%s", usage)
		return 2
	}
	oldText, err := os.ReadFile(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	newText, err := os.ReadFile(pos[1])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	res := configdiff.Diff(string(oldText), string(newText))
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printConfigDiff(stdout, res)
	return 0
}

func printConfigDiff(stdout io.Writer, r configdiff.Result) {
	fmt.Fprintf(stdout, "%d added, %d removed", r.Added, r.Removed)
	if len(r.Sections) > 0 {
		fmt.Fprintf(stdout, " — changed sections: %s", strings.Join(r.Sections, ", "))
	}
	fmt.Fprint(stdout, "\n\n")
	for _, l := range r.Lines {
		switch l.Op {
		case configdiff.OpAdd:
			fmt.Fprintf(stdout, "+ %s\n", l.Text)
		case configdiff.OpDel:
			fmt.Fprintf(stdout, "- %s\n", l.Text)
		}
	}
}

// ---- breaches --------------------------------------------------------------

func runBreaches(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("breaches", flag.ContinueOnError)
	fs.SetOutput(stderr)
	latest := fs.Bool("latest", false, "show the most recent breach HIBP knows about")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	client := &http.Client{Timeout: 20 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if *latest {
		b, err := hibp.LatestBreach(ctx, client)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, b)
		}
		printBreach(stdout, b)
		return 0
	}

	if len(pos) != 1 {
		fmt.Fprintf(stderr, "breaches needs an email address (or --latest)\n%s", usage)
		return 2
	}
	names, err := xon.CheckEmail(ctx, client, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, names)
	}
	printBreachNames(stdout, pos[0], names)
	return 0
}

func printBreachNames(stdout io.Writer, email string, names []string) {
	if len(names) == 0 {
		fmt.Fprintf(stdout, "%s — no breaches found (XposedOrNot)\n", email)
		return
	}
	fmt.Fprintf(stdout, "%s — found in %d breach(es):\n", email, len(names))
	for _, n := range names {
		fmt.Fprintf(stdout, "  %s\n", n)
	}
	fmt.Fprintln(stdout, "\nNote: an email lookup sends the full address to XposedOrNot (no k-anonymity).")
}

func printBreach(stdout io.Writer, b hibp.Breach) {
	reportRow(stdout, "Breach", b.Title)
	reportRow(stdout, "Domain", b.Domain)
	reportRow(stdout, "Date", b.BreachDate)
	reportRow(stdout, "Accounts", fmt.Sprintf("%d", b.PwnCount))
	if len(b.DataClasses) > 0 {
		reportRow(stdout, "Exposed", strings.Join(b.DataClasses, ", "))
	}
	if b.Description != "" {
		fmt.Fprintf(stdout, "\n%s\n", firstLine(b.Description, 200))
	}
}

// readFileOrStdin returns the first positional's file contents, or all of
// stdin when no positional is given.
func readFileOrStdin(pos []string, stdin io.Reader) ([]byte, error) {
	if len(pos) >= 1 {
		return os.ReadFile(pos[0])
	}
	return io.ReadAll(stdin)
}
