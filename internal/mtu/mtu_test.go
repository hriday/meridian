package mtu

import (
	"context"
	"testing"
)

// thresholdProber fits any payload <= limit. If nextHopAt > 0, an over-limit
// probe reports that next-hop MTU (simulating a cooperative router).
type thresholdProber struct {
	limit     int
	nextHopAt int
	calls     int
}

func (tp *thresholdProber) Probe(ctx context.Context, payload int) (ProbeResult, error) {
	tp.calls++
	if payload <= tp.limit {
		return ProbeResult{Outcome: Fits}, nil
	}
	if tp.nextHopAt > 0 {
		return ProbeResult{Outcome: TooBig, NextHop: tp.nextHopAt}, nil
	}
	return ProbeResult{Outcome: Unknown}, nil
}
func (tp *thresholdProber) Close() error { return nil }

func TestDiscoverFindsBoundary(t *testing.T) {
	tp := &thresholdProber{limit: 1372} // path MTU 1400 => max payload 1372
	res, err := Discover(context.Background(), tp, "example.com", "203.0.113.5", 1500, 1472, 1)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !res.DFEnforced {
		t.Fatal("DF should read as enforced when the over-MTU probe is rejected")
	}
	if res.PayloadMax != 1372 || res.PathMTU != 1400 {
		t.Errorf("PayloadMax=%d PathMTU=%d, want 1372/1400", res.PayloadMax, res.PathMTU)
	}
	if !res.Bottleneck {
		t.Errorf("1400 path under a 1500 local link should be a bottleneck")
	}
}

func TestDiscoverNoBottleneckWhenFull(t *testing.T) {
	tp := &thresholdProber{limit: 1472} // full 1500 path
	res, err := Discover(context.Background(), tp, "h", "1.2.3.4", 1500, 1472, 1)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res.PathMTU != 1500 || res.Bottleneck {
		t.Errorf("PathMTU=%d bottleneck=%v, want 1500/false", res.PathMTU, res.Bottleneck)
	}
}

func TestDiscoverDetectsDFNotEnforced(t *testing.T) {
	// A kernel that fragments everything: even a packet over the local MTU
	// "fits". The tool must refuse to report a bogus MTU.
	tp := &thresholdProber{limit: 100000}
	res, err := Discover(context.Background(), tp, "h", "1.2.3.4", 1500, 1472, 1)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res.DFEnforced {
		t.Fatal("an over-MTU packet that gets through means DF is NOT enforced")
	}
	if res.PathMTU != 0 {
		t.Errorf("no MTU should be reported when DF isn't enforced, got %d", res.PathMTU)
	}
	if res.Note == "" {
		t.Error("a DF-not-enforced result must explain why")
	}
}

func TestDiscoverUsesNextHopMTU(t *testing.T) {
	tp := &thresholdProber{limit: 1372, nextHopAt: 1400}
	res, err := Discover(context.Background(), tp, "h", "1.2.3.4", 1500, 9000, 0)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if res.PathMTU != 1400 || res.NextHopMTU != 1400 {
		t.Errorf("PathMTU=%d NextHopMTU=%d, want 1400/1400", res.PathMTU, res.NextHopMTU)
	}
}

func TestDiscoverBaselineFailureErrors(t *testing.T) {
	// limit between the over-MTU self-check size and 0: self-check passes
	// (over-MTU rejected) but nothing else replies, so baseline fails.
	tp := &thresholdProber{limit: -1}
	_, err := Discover(context.Background(), tp, "h", "1.2.3.4", 1500, 1472, 1)
	if err == nil {
		t.Fatal("a host that answers nothing should error, not report an MTU")
	}
}

func TestDiscoverNoLocalMTUSkipsSelfCheck(t *testing.T) {
	// localMTU 0 (unknown): self-check is skipped, measurement proceeds.
	tp := &thresholdProber{limit: 1372}
	res, err := Discover(context.Background(), tp, "h", "1.2.3.4", 0, 1472, 1)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !res.DFEnforced || res.PathMTU != 1400 {
		t.Errorf("with unknown local MTU, want DFEnforced + PathMTU 1400, got %v/%d", res.DFEnforced, res.PathMTU)
	}
}
