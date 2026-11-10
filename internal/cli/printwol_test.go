package cli

import (
	"bytes"
	"testing"
)

func TestRunWOLMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runWOL(nil, &out, &errb); code != 2 {
		t.Errorf("runWOL with no args: got exit %d, want 2", code)
	}
}

func TestRunWOLBadMAC(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runWOL([]string{"nope"}, &out, &errb); code != 1 {
		t.Errorf("runWOL with a bad MAC: got exit %d, want 1", code)
	}
}
