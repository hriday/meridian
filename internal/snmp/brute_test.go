package snmp

import (
	"context"
	"testing"
	"time"
)

func TestBruteCommunitiesFindsWorking(t *testing.T) {
	orig := communityProbe
	defer func() { communityProbe = orig }()
	communityProbe = func(_ string, community string, _ time.Duration) bool {
		return community == "private" || community == "cisco"
	}
	got, err := BruteCommunities(context.Background(), "10.0.0.1",
		[]string{"public", "private", "public", "admin", "cisco"}, time.Second, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "private" || got[1] != "cisco" {
		t.Errorf("got %v, want [private cisco] in input order, deduped", got)
	}
}

func TestBruteCommunitiesValidation(t *testing.T) {
	if _, err := BruteCommunities(context.Background(), "10.0.0.1", nil, time.Second, 2); err == nil {
		t.Error("want an error for no communities")
	}
	if _, err := BruteCommunities(context.Background(), "  ", []string{"public"}, time.Second, 2); err == nil {
		t.Error("want an error for an empty device")
	}
}

func TestBruteCommunitiesNoneWork(t *testing.T) {
	orig := communityProbe
	defer func() { communityProbe = orig }()
	communityProbe = func(string, string, time.Duration) bool { return false }
	got, err := BruteCommunities(context.Background(), "10.0.0.1", []string{"public", "private"}, time.Second, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("want no working communities, got %v", got)
	}
}
