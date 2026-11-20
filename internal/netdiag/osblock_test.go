package netdiag

import (
	"context"
	"net"
	"os"
	"syscall"
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

func TestOwnReach_RefusedCountsAsReached(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close() // nothing listens now: connect is refused
	ok, ran := tcpReach(context.Background(), "127.0.0.1", port)
	if !ok || !ran {
		t.Fatalf("refused connection: ok=%v ran=%v, want reached", ok, ran)
	}
	if !isRefused(&net.OpError{Err: os.NewSyscallError("connect", syscall.ECONNREFUSED)}) {
		t.Error("wrapped ECONNREFUSED must classify as refused")
	}
}

func TestPlainIPv4Peers_DropsEverythingButIPv4(t *testing.T) {
	got := plainIPv4Peers([]string{"192.168.1.2", "fe80::1%x;calc.exe", "2001:db8::1", "x; calc", "10.0.0.1"})
	if len(got) != 2 || got[0] != "192.168.1.2" || got[1] != "10.0.0.1" {
		t.Fatalf("got %v", got)
	}
}
