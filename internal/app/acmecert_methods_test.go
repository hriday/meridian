package app

import (
	"strings"
	"testing"
)

func TestSplitDomainList(t *testing.T) {
	got := splitDomainList("example.com, www.example.com\n*.internal.test  ;  api.example.com")
	want := []string{"example.com", "www.example.com", "*.internal.test", "api.example.com"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(splitDomainList("   ,  ;  ")) != 0 {
		t.Error("blank input should yield no domains")
	}
}

func TestCertFinalizeUnknownOrder(t *testing.T) {
	a := New()
	if _, err := a.CertFinalize("nope"); err == nil {
		t.Error("finalizing an unknown order should error, not panic")
	}
	a.CertCancel("nope") // must be a safe no-op
}

func TestNewOrderIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id := newOrderID()
		if id == "" || seen[id] {
			t.Fatalf("order id collision or empty: %q", id)
		}
		seen[id] = true
	}
}
