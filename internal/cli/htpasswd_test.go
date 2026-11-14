package cli

import (
	"bytes"
	"strings"
	"testing"
)

func withStdin(s string, fn func()) {
	old := htpasswdStdin
	htpasswdStdin = strings.NewReader(s)
	defer func() { htpasswdStdin = old }()
	fn()
}

func TestHtpasswdGenerateAndCheck(t *testing.T) {
	var out, errb bytes.Buffer
	withStdin("hunter2\n", func() {
		if code := runHtpasswd([]string{"admin", "--cost", "4"}, &out, &errb); code != 0 {
			t.Fatalf("generate exit %d: %s", code, errb.String())
		}
	})
	line := strings.TrimSpace(out.String())
	if !strings.HasPrefix(line, "admin:$2") {
		t.Fatalf("expected an htpasswd line, got %q", line)
	}
	hash := strings.SplitN(line, ":", 2)[1]

	// Verify the right password matches.
	var o2, e2 bytes.Buffer
	withStdin("hunter2", func() {
		if code := runHtpasswd([]string{"--check", hash}, &o2, &e2); code != 0 {
			t.Errorf("check of correct password exit %d: %s", code, o2.String())
		}
	})
	if !strings.Contains(o2.String(), "MATCH") {
		t.Errorf("correct password should MATCH: %q", o2.String())
	}

	// Wrong password fails (exit 1).
	var o3, e3 bytes.Buffer
	withStdin("wrong", func() {
		if code := runHtpasswd([]string{"--check", hash}, &o3, &e3); code != 1 {
			t.Errorf("wrong password should exit 1, got %d", code)
		}
	})
}

func TestHtpasswdNoPassword(t *testing.T) {
	var o, e bytes.Buffer
	withStdin("", func() {
		if code := runHtpasswd([]string{"admin"}, &o, &e); code != 2 {
			t.Errorf("no password should exit 2, got %d", code)
		}
	})
}
