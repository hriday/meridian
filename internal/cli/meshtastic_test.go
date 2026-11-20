package cli

import (
	"bytes"
	"strings"
	"testing"
)

// A failed read must exit non-zero and print nothing to stdout, even in --json
// mode — never a zero-value snapshot dressed up as success. A non-existent
// serial device fails fast in open(), with no network involved.
func TestRunMeshtastic_ErrorExitsNonZeroEvenWithJSON(t *testing.T) {
	var out, errb bytes.Buffer
	code := runMeshtastic([]string{"--json", "/dev/nonexistent-meridian-xyz-0"}, &out, &errb)

	if code == 0 {
		t.Fatalf("exit code = 0 on error; want non-zero (stdout=%q)", out.String())
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("stdout should be empty on error, got %q", out.String())
	}
	if errb.Len() == 0 {
		t.Error("expected an error message on stderr")
	}
}
