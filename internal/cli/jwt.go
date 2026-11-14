package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/jwtinspect"
)

func runJWT(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("jwt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	secret := fs.String("secret", "", "HMAC secret to verify an HS* token")
	keyfile := fs.String("keyfile", "", "PEM public key / certificate file to verify an RS*/ES*/PS*/EdDSA token")
	asJSON := fs.Bool("json", false, "output the decoded token as JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "jwt needs exactly one token\n%s", usage)
		return 2
	}

	d, err := jwtinspect.Decode(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	key := *secret
	if *keyfile != "" {
		b, err := os.ReadFile(*keyfile)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		key = string(b)
	}
	var vr *jwtinspect.VerifyResult
	if *secret != "" || *keyfile != "" {
		r, err := jwtinspect.Verify(pos[0], key)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		vr = &r
	}

	if *asJSON {
		return writeJSON(stdout, stderr, struct {
			jwtinspect.Decoded
			Verify *jwtinspect.VerifyResult `json:"verify,omitempty"`
		}{d, vr})
	}
	printJWT(stdout, d, vr)
	return 0
}

func printJWT(w io.Writer, d jwtinspect.Decoded, vr *jwtinspect.VerifyResult) {
	reportRow(w, "Algorithm", d.Alg)
	if d.Typ != "" {
		reportRow(w, "Type", d.Typ)
	}
	if d.Kid != "" {
		reportRow(w, "Key ID (kid)", d.Kid)
	}
	if d.Issuer != "" {
		reportRow(w, "Issuer (iss)", d.Issuer)
	}
	if d.Subject != "" {
		reportRow(w, "Subject (sub)", d.Subject)
	}
	if len(d.Audience) > 0 {
		reportRow(w, "Audience (aud)", strings.Join(d.Audience, ", "))
	}
	if d.IssuedAt != nil {
		reportRow(w, "Issued (iat)", d.IssuedAt.Format(time.RFC3339))
	}
	if d.NotBefore != nil {
		reportRow(w, "Not before (nbf)", d.NotBefore.Format(time.RFC3339))
	}
	if d.ExpiresAt != nil {
		status := "valid"
		if d.Expired {
			status = "EXPIRED"
		}
		reportRow(w, "Expires (exp)", fmt.Sprintf("%s (%s)", d.ExpiresAt.Format(time.RFC3339), status))
	}
	for _, warn := range d.Warnings {
		fmt.Fprintf(w, "⚠ %s\n", warn)
	}

	fmt.Fprintf(w, "\nHeader:\n%s\n", d.HeaderJSON)
	fmt.Fprintf(w, "\nPayload:\n%s\n", d.PayloadJSON)

	if vr != nil {
		mark := "✗ INVALID"
		if vr.Valid {
			mark = "✓ VALID"
		}
		fmt.Fprintf(w, "\nSignature (%s): %s — %s\n", vr.Algorithm, mark, vr.Reason)
	}
}
