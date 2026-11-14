package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunTranscodeArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runTranscode([]string{"base64-encode", "Meridian"}, &o, &e); code != 0 {
		t.Fatalf("exit %d: %s", code, e.String())
	}
	if strings.TrimSpace(o.String()) != "TWVyaWRpYW4=" {
		t.Errorf("got %q", o.String())
	}
}

func TestRunTranscodeStdin(t *testing.T) {
	old := transcodeStdin
	transcodeStdin = strings.NewReader("TWVyaWRpYW4=")
	defer func() { transcodeStdin = old }()
	var o, e bytes.Buffer
	if code := runTranscode([]string{"base64-decode"}, &o, &e); code != 0 {
		t.Fatalf("exit %d: %s", code, e.String())
	}
	if strings.TrimSpace(o.String()) != "Meridian" {
		t.Errorf("stdin decode got %q", o.String())
	}
}

func TestRunTranscodeNoOp(t *testing.T) {
	var o, e bytes.Buffer
	if code := runTranscode(nil, &o, &e); code != 2 {
		t.Errorf("no op should exit 2, got %d", code)
	}
}
