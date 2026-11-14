package cli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/hriday/meridian/internal/totp"
)

func runTOTP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("totp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "totp needs one base32 secret or otpauth:// URI\n%s", usage)
		return 2
	}
	cfg, err := totp.Parse(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	code, err := totp.Generate(cfg, time.Now().Unix())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, code)
	}
	printTOTP(stdout, code)
	return 0
}

func printTOTP(w io.Writer, c totp.Code) {
	if c.Account != "" || c.Issuer != "" {
		reportRow(w, "Account", fmt.Sprintf("%s %s", c.Issuer, c.Account))
	}
	reportRow(w, "Code", c.Code)
	reportRow(w, "Valid for", fmt.Sprintf("%d more second(s) (of %d)", c.SecondsLeft, c.Period))
}
