package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/webconf"
)

// webStdin is the config source when no file argument is given; tests override it.
var webStdin io.Reader = os.Stdin

func runWebAnalyze(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("webanalyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	server := fs.String("server", "nginx", "web server type (nginx)")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}

	var data []byte
	var err error
	if len(pos) >= 1 {
		data, err = os.ReadFile(pos[0])
	} else {
		data, err = io.ReadAll(webStdin)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	a, err := webconf.Analyze(string(data), *server)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, a)
	}
	printWebSummary(stdout, a)
	return 0
}

func printWebSummary(stdout io.Writer, a webconf.Analysis) {
	fmt.Fprintf(stdout, "%s — %d server blocks, %d listeners, %d upstreams, %d locations, %d errors, %d warnings\n",
		a.ServerType, a.Counts.Servers, a.Counts.Listeners, a.Counts.Upstreams, a.Counts.Locations, a.Counts.Errors, a.Counts.Warnings)

	for _, s := range a.Servers {
		name := strings.Join(s.Names, " ")
		if name == "" {
			name = "(unnamed)"
		}
		fmt.Fprintf(stdout, "\nserver %s", name)
		for _, l := range s.Listeners {
			fmt.Fprintf(stdout, "  [:%d%s]", l.Port, sslTag(l))
		}
		fmt.Fprintln(stdout)
		for _, loc := range s.Locations {
			if loc.Target != "" {
				fmt.Fprintf(stdout, "    %s → %s (%s)\n", loc.Path, loc.Target, loc.Kind)
			}
		}
	}
	if len(a.Findings) > 0 {
		fmt.Fprintln(stdout, "\nfindings:")
		for _, f := range a.Findings {
			where := f.File
			if f.Line > 0 {
				where = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			fmt.Fprintf(stdout, "  [%s/%s] %s  (%s)\n", f.Severity, f.Category, f.Title, where)
		}
	}
}

func sslTag(l webconf.Listener) string {
	t := ""
	if l.SSL {
		t += " ssl"
	}
	if l.HTTP2 {
		t += " http2"
	}
	return t
}
