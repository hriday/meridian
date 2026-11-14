package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/totp"
)

func TestPrintTOTP(t *testing.T) {
	var buf bytes.Buffer
	printTOTP(&buf, totp.Code{Code: "287082", SecondsLeft: 12, Period: 30, Issuer: "GitHub", Account: "alice"})
	out := buf.String()
	for _, want := range []string{"287082", "12 more second", "GitHub"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunTOTPLive(t *testing.T) {
	var o, e bytes.Buffer
	// a valid base32 secret should produce a 6-digit code
	if code := runTOTP([]string{"GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"}, &o, &e); code != 0 {
		t.Fatalf("exit %d: %s", code, e.String())
	}
	if !strings.Contains(o.String(), "Code") {
		t.Errorf("output should contain a code: %s", o.String())
	}
}

func TestRunTOTPMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runTOTP(nil, &o, &e); code != 2 {
		t.Errorf("no args: got exit %d, want 2", code)
	}
}
