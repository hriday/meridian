package netdiag

import (
	"context"
	"testing"
)

func fixed(ok, ran bool) reachFn {
	return func(context.Context, string, int) (bool, bool) { return ok, ran }
}

func TestOSBlock_BaselineReachesOwnCannot_Blocked(t *testing.T) {
	st, detail := osBlockCheck(context.Background(), []string{"192.168.1.10"}, fixed(false, true), fixed(true, true))
	if st != OSBlockBlocked {
		t.Fatalf("got %v, want OSBlockBlocked (%s)", st, detail)
	}
	if detail == "" {
		t.Error("expected a detail string")
	}
}

func TestOSBlock_BothFail_NotBlocked(t *testing.T) {
	st, _ := osBlockCheck(context.Background(), []string{"192.168.1.10"}, fixed(false, true), fixed(false, true))
	if st != OSBlockNotBlocked {
		t.Fatalf("got %v, want OSBlockNotBlocked", st)
	}
}

func TestOSBlock_BaselineCouldNotRun_CouldNotDetermine(t *testing.T) {
	st, _ := osBlockCheck(context.Background(), []string{"192.168.1.10"}, fixed(false, true), fixed(false, false))
	if st != OSBlockCouldNotDetermine {
		t.Fatalf("got %v, want OSBlockCouldNotDetermine", st)
	}
}
