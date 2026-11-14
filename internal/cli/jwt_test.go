package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/jwtinspect"
)

func TestPrintJWTDecoded(t *testing.T) {
	var buf bytes.Buffer
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	printJWT(&buf, jwtinspect.Decoded{
		Alg: "HS256", Typ: "JWT", Kid: "k1", Issuer: "meridian", Subject: "42",
		ExpiresAt: &exp, HeaderJSON: "{header}", PayloadJSON: "{payload}",
		Warnings: []string{"token is expired"},
	}, nil)
	out := buf.String()
	for _, want := range []string{"HS256", "k1", "meridian", "Expires", "⚠ token is expired", "Header:", "Payload:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintJWTWithVerify(t *testing.T) {
	var buf bytes.Buffer
	printJWT(&buf, jwtinspect.Decoded{Alg: "HS256", HeaderJSON: "{}", PayloadJSON: "{}"},
		&jwtinspect.VerifyResult{Algorithm: "HS256", Valid: true, Reason: "signature is valid"})
	if !strings.Contains(buf.String(), "✓ VALID") {
		t.Errorf("a valid signature should print VALID: %s", buf.String())
	}
}

func TestRunJWTMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runJWT(nil, &o, &e); code != 2 {
		t.Errorf("jwt with no args: got exit %d, want 2", code)
	}
}
