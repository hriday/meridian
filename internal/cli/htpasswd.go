package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hriday/meridian/internal/pwhash"
)

// htpasswdStdin is where the password is read from (a seam for tests). Reading
// from stdin keeps the password out of the argument list / shell history.
var htpasswdStdin io.Reader = os.Stdin

func runHtpasswd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("htpasswd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cost := fs.Int("cost", pwhash.DefaultCost, "bcrypt cost (4-31; higher is slower/stronger)")
	check := fs.String("check", "", "verify the password against this bcrypt hash instead of generating")
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}

	pwBytes, err := io.ReadAll(htpasswdStdin)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	password := strings.TrimRight(string(pwBytes), "\r\n")
	if password == "" {
		fmt.Fprintln(stderr, "no password on stdin — pipe the password in, e.g.  printf secret | meridian-cli htpasswd admin")
		return 2
	}

	if *check != "" {
		res, err := pwhash.Check(password, *check)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, res)
		}
		if res.Match {
			fmt.Fprintln(stdout, "MATCH — the password matches the hash")
			return 0
		}
		fmt.Fprintln(stdout, "NO MATCH")
		return 1
	}

	user := ""
	if len(pos) >= 1 {
		user = pos[0]
	}
	res, err := pwhash.Generate(password, user, *cost)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	if res.Htpasswd != "" {
		fmt.Fprintln(stdout, res.Htpasswd)
	} else {
		fmt.Fprintln(stdout, res.Hash)
	}
	return 0
}
