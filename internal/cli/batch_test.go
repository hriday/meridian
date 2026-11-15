package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// withBatchStdin runs fn with the batch target list coming from s.
func withBatchStdin(s string, fn func()) {
	old := batchStdin
	batchStdin = strings.NewReader(s)
	defer func() { batchStdin = old }()
	fn()
}

func TestBatchJSONNDJSON(t *testing.T) {
	var out bytes.Buffer
	var code int
	withBatchStdin("1.1.1.1\n# a comment\n\n8.8.8.8\n", func() {
		code = runBatch([]string{"classify", "--json"}, &out, io.Discard)
	})
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 { // two targets; blank + comment skipped
		t.Fatalf("want 2 NDJSON lines, got %d: %q", len(lines), out.String())
	}
	for _, ln := range lines {
		if !strings.HasPrefix(ln, "{") {
			t.Fatalf("line isn't JSON: %q", ln)
		}
	}
}

func TestBatchTextSections(t *testing.T) {
	var out bytes.Buffer
	withBatchStdin("1.1.1.1\n10.0.0.0/8\n", func() {
		runBatch([]string{"classify"}, &out, io.Discard)
	})
	s := out.String()
	if !strings.Contains(s, "== 1.1.1.1 ==") || !strings.Contains(s, "== 10.0.0.0/8 ==") {
		t.Fatalf("missing per-target sections: %q", s)
	}
}

func TestBatchNoTargets(t *testing.T) {
	var out, errb bytes.Buffer
	var code int
	withBatchStdin("\n  \n", func() {
		code = runBatch([]string{"classify"}, &out, &errb)
	})
	if code != 2 {
		t.Fatalf("want exit 2 for empty stdin, got %d", code)
	}
}

func TestBatchRejectsRecursion(t *testing.T) {
	var out, errb bytes.Buffer
	code := runBatch([]string{"batch"}, &out, &errb)
	if code != 2 {
		t.Fatalf("want exit 2 for `batch batch`, got %d", code)
	}
}
