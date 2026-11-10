package cli

import (
	"bytes"
	"strings"
	"testing"
)

const fgCLISample = `config system interface
    edit "port1.30"
        set vlanid 30
    next
end
config firewall policy
    edit 1
        set srcintf "port1.30"
        set dstintf "port1.20"
        set service "HTTPS"
        set action accept
    next
end`

func TestRunFwAnalyzeStdinJSON(t *testing.T) {
	old := fwStdin
	fwStdin = strings.NewReader(fgCLISample)
	defer func() { fwStdin = old }()

	var out, errb bytes.Buffer
	code := runFwAnalyze([]string{"--json"}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errb.String())
	}
	s := out.String()
	for _, want := range []string{`"vlans"`, `"port1.30"`, `"flows"`, `"port1.20"`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON output missing %q\n%s", want, s)
		}
	}
}

func TestRunFwAnalyzeTextSummary(t *testing.T) {
	old := fwStdin
	fwStdin = strings.NewReader(fgCLISample)
	defer func() { fwStdin = old }()

	var out, errb bytes.Buffer
	code := runFwAnalyze(nil, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "port1.30") || !strings.Contains(s, "port1.20") {
		t.Errorf("text summary missing flow endpoints\n%s", s)
	}
}

func TestRunFwAnalyzeEmptyStdin(t *testing.T) {
	old := fwStdin
	fwStdin = strings.NewReader("")
	defer func() { fwStdin = old }()

	var out, errb bytes.Buffer
	if code := runFwAnalyze(nil, &out, &errb); code != 1 {
		t.Errorf("empty config: got exit %d, want 1", code)
	}
}
