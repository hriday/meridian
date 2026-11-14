package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/transcode"
)

// transcodeStdin is a seam for tests.
var transcodeStdin io.Reader = os.Stdin

func runTranscode(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("transcode", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) < 1 {
		fmt.Fprintf(stderr, "transcode needs an operation: %s\n", strings.Join(transcode.Ops(), ", "))
		return 2
	}
	op := pos[0]

	var input string
	if len(pos) >= 2 {
		input = strings.Join(pos[1:], " ")
	} else {
		b, err := io.ReadAll(transcodeStdin)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		input = strings.TrimRight(string(b), "\n")
	}

	out, err := transcode.Apply(op, input)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, out)
	return 0
}
