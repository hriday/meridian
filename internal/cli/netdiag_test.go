package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/netdiag"
)

func fakeNetDiag(t *testing.T, r netdiag.Report) {
	t.Helper()
	old := netdiagRun
	netdiagRun = func(ctx context.Context, o netdiag.Options, f func(netdiag.StepResult)) (netdiag.Report, error) {
		return r, nil
	}
	t.Cleanup(func() { netdiagRun = old })
}

func TestRunNetDiag_JSON(t *testing.T) {
	fakeNetDiag(t, netdiag.Report{Verdict: netdiag.VerdictClientIsolation, Statement: "isolated"})
	var out, errb bytes.Buffer
	code := runNetDiag([]string{"--json"}, &out, &errb)
	if code != 1 || !strings.Contains(out.String(), `"client_isolation"`) {
		t.Fatalf("code=%d out=%s err=%s", code, out.String(), errb.String())
	}
}

func TestRunNetDiag_HumanListsFix(t *testing.T) {
	fakeNetDiag(t, netdiag.Report{
		Verdict: netdiag.VerdictOSBlockingMeridian, Statement: "OS blocks Meridian",
		Fixes: []string{"Grant Local Network access"},
		Steps: []netdiag.StepResult{{Name: "gateway", Status: "pass"}, {Name: "peers", Status: "fail", Detail: "refused"}},
	})
	var out, errb bytes.Buffer
	code := runNetDiag(nil, &out, &errb)
	s := out.String()
	if code != 1 || !strings.Contains(s, "Grant Local Network access") || !strings.Contains(s, "OS blocks Meridian") || !strings.Contains(s, "peers") {
		t.Fatalf("code=%d out=%s", code, s)
	}
}

func TestRunNetDiag_ExitCodes(t *testing.T) {
	fakeNetDiag(t, netdiag.Report{Verdict: netdiag.VerdictAllClear})
	var out, errb bytes.Buffer
	if c := runNetDiag(nil, &out, &errb); c != 0 {
		t.Fatalf("all_clear code=%d", c)
	}
	if c := runNetDiag([]string{"a", "b"}, &out, &errb); c != 2 {
		t.Fatalf("usage code=%d", c)
	}
}
