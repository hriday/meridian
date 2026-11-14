package pwhash

import (
	"strings"
	"testing"
)

func TestGenerateAndCheck(t *testing.T) {
	g, err := Generate("hunter2", "admin", 4) // low cost = fast test
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.HasPrefix(g.Hash, "$2") {
		t.Errorf("hash should be bcrypt: %q", g.Hash)
	}
	if g.Htpasswd != "admin:"+g.Hash {
		t.Errorf("htpasswd line wrong: %q", g.Htpasswd)
	}

	ok, err := Check("hunter2", g.Hash)
	if err != nil || !ok.Match {
		t.Errorf("correct password should match (err=%v)", err)
	}
	bad, err := Check("wrong", g.Hash)
	if err != nil || bad.Match {
		t.Errorf("wrong password must not match (err=%v)", err)
	}
}

func TestCheckAcceptsHtpasswdLine(t *testing.T) {
	g, _ := Generate("pw", "bob", 4)
	ok, err := Check("pw", g.Htpasswd) // pass the full "bob:$2..." line
	if err != nil || !ok.Match {
		t.Errorf("htpasswd line should verify (err=%v match=%v)", err, ok.Match)
	}
}

func TestCheckRejectsNonBcrypt(t *testing.T) {
	if _, err := Check("pw", "5f4dcc3b5aa765d61d8327deb882cf99"); err == nil {
		t.Error("a non-bcrypt hash should error")
	}
}

func TestGenerateValidatesCost(t *testing.T) {
	if _, err := Generate("pw", "", 99); err == nil {
		t.Error("an out-of-range cost should error")
	}
	if _, err := Generate("", "", 0); err == nil {
		t.Error("an empty password should error")
	}
}
