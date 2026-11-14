package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/sipping"
)

func TestPrintSIP(t *testing.T) {
	var buf bytes.Buffer
	printSIP(&buf, sipping.Result{
		Target: "pbx.example.com:5060", Transport: "udp", RTTMillis: 14,
		StatusCode: 200, Reason: "OK", Server: "Asterisk 18",
		Allow: []string{"INVITE", "OPTIONS", "BYE"}, Supported: []string{"timer"},
	})
	out := buf.String()
	for _, want := range []string{"pbx.example.com:5060", "UDP", "200 OK", "reachable and healthy", "Asterisk 18", "INVITE"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunSIPMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runSIP(nil, &o, &e); code != 2 {
		t.Errorf("no args: got exit %d, want 2", code)
	}
}
