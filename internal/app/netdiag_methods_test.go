package app

import (
	"context"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/netdiag"
	"github.com/stretchr/testify/require"
)

func TestNetworkDiagnostic_DelegatesToEngine(t *testing.T) {
	want := netdiag.Report{
		Verdict:   netdiag.Verdict("os_block"),
		Statement: "known",
		Causes:    []string{"c"},
		Fixes:     []string{"f"},
		Steps:     []netdiag.StepResult{{Name: "n", Status: "pass", Detail: "d"}},
	}
	old := netdiagRun
	t.Cleanup(func() { netdiagRun = old })

	var gotTarget string
	var gotDeadline bool
	var gotRemaining time.Duration
	var gotOnStep func(netdiag.StepResult)
	netdiagRun = func(ctx context.Context, o netdiag.Options, onStep func(netdiag.StepResult)) (netdiag.Report, error) {
		gotTarget = o.Target
		var dl time.Time
		dl, gotDeadline = ctx.Deadline()
		gotRemaining = time.Until(dl)
		gotOnStep = onStep
		return want, nil
	}

	got, err := (&App{}).NetworkDiagnostic("192.168.1.10")
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, "192.168.1.10", gotTarget)
	require.True(t, gotDeadline, "ctx must be bounded")
	require.Greater(t, gotRemaining, time.Second)
	require.LessOrEqual(t, gotRemaining, 30*time.Second)
	require.Nil(t, gotOnStep)
}
