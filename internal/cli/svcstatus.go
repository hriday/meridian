package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/svcstatus"
)

func runServiceStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "status needs one service or status-page URL (try: %s)\n%s",
			strings.Join(svcstatus.KnownSlugs(), ", "), usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s, err := svcstatus.Check(ctx, &http.Client{Timeout: 12 * time.Second}, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, s)
	}
	printServiceStatus(stdout, s)
	return 0
}

func printServiceStatus(stdout io.Writer, s svcstatus.Status) {
	mark := "OUTAGE"
	if s.Operational {
		mark = "OK"
	}
	fmt.Fprintf(stdout, "[%s] %s — %s\n", mark, s.Service, s.Description)
	if len(s.Impacted) > 0 {
		fmt.Fprintln(stdout, "\nAffected components:")
		for _, c := range s.Impacted {
			fmt.Fprintf(stdout, "  %-30s %s\n", c.Name, strings.ReplaceAll(c.Status, "_", " "))
		}
	}
	if len(s.Incidents) > 0 {
		fmt.Fprintln(stdout, "\nActive incidents:")
		for _, in := range s.Incidents {
			fmt.Fprintf(stdout, "  [%s] %s (%s)\n", in.Impact, in.Name, in.Status)
			if in.Shortlink != "" {
				fmt.Fprintf(stdout, "        %s\n", in.Shortlink)
			}
		}
	}
}
