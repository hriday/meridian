package mtr

import (
	"context"
	"testing"
	"time"

	"github.com/hriday/meridian/internal/trace"
)

// fakeSweeper returns a scripted trace.Result per round and records how many
// times it was called.
func fakeSweeper(rounds [][]trace.Hop, addr string) (Sweeper, *int) {
	var call int
	s := func(ctx context.Context, target string, useV6 bool, onHop func(trace.Hop)) (trace.Result, error) {
		hs := rounds[call]
		call++
		for _, h := range hs {
			onHop(h)
		}
		return trace.Result{Target: target, Addr: addr, Hops: hs, Reached: true}, nil
	}
	return s, &call
}

func TestMonitorAccumulatesPerHopStats(t *testing.T) {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	rounds := [][]trace.Hop{
		{
			{TTL: 1, Addr: "10.0.0.1", Host: "gw", RTTs: []time.Duration{ms(2)}},
			{TTL: 2, Addr: "203.0.113.5", ASN: "AS64500", RTTs: []time.Duration{ms(10)}},
		},
		{
			{TTL: 1, Addr: "10.0.0.1", Host: "gw", RTTs: []time.Duration{ms(4)}},
			{TTL: 2, Addr: "203.0.113.5", Timeout: true},
		},
	}
	sweep, calls := fakeSweeper(rounds, "203.0.113.5")

	ctx, cancel := context.WithCancel(context.Background())
	var updates []Update
	err := Monitor(ctx, sweep, "example.com", false, 0, func(u Update) {
		updates = append(updates, u)
		if u.Round == 2 {
			cancel()
		}
	})
	if err != nil {
		t.Fatalf("Monitor returned error: %v", err)
	}
	if *calls != 2 {
		t.Fatalf("sweep called %d times, want 2 (must stop after ctx cancel)", *calls)
	}
	if len(updates) != 2 {
		t.Fatalf("got %d updates, want 2", len(updates))
	}

	last := updates[1]
	if len(last.Hops) != 2 {
		t.Fatalf("final update has %d hops, want 2", len(last.Hops))
	}

	h1 := last.Hops[0]
	if h1.TTL != 1 || h1.Sent != 2 || h1.Recv != 2 || h1.LossPct != 0 {
		t.Errorf("hop1: got TTL=%d Sent=%d Recv=%d Loss=%.0f, want 1/2/2/0", h1.TTL, h1.Sent, h1.Recv, h1.LossPct)
	}
	if h1.Best != ms(2) || h1.Worst != ms(4) || h1.Last != ms(4) || h1.Avg != ms(3) {
		t.Errorf("hop1 rtts: best=%v worst=%v last=%v avg=%v, want 2/4/4/3ms", h1.Best, h1.Worst, h1.Last, h1.Avg)
	}
	if h1.Host != "gw" {
		t.Errorf("hop1 host = %q, want gw", h1.Host)
	}

	h2 := last.Hops[1]
	if h2.TTL != 2 || h2.Sent != 2 || h2.Recv != 1 {
		t.Errorf("hop2: got TTL=%d Sent=%d Recv=%d, want 2/2/1", h2.TTL, h2.Sent, h2.Recv)
	}
	if h2.LossPct != 50 {
		t.Errorf("hop2 loss = %.1f, want 50", h2.LossPct)
	}
	if h2.Best != ms(10) || h2.Worst != ms(10) || h2.Last != ms(10) {
		t.Errorf("hop2 rtts: best=%v worst=%v last=%v, want all 10ms (timeout round ignored for RTT)", h2.Best, h2.Worst, h2.Last)
	}
	if h2.ASN != "AS64500" {
		t.Errorf("hop2 ASN = %q, want AS64500 (retained across rounds)", h2.ASN)
	}
}

func TestMonitorTransientErrorPreservesState(t *testing.T) {
	ms := func(n int) time.Duration { return time.Duration(n) * time.Millisecond }
	var call int
	sweep := func(ctx context.Context, target string, useV6 bool, onHop func(trace.Hop)) (trace.Result, error) {
		call++
		if call == 2 {
			return trace.Result{}, context.DeadlineExceeded // a single lost sweep
		}
		h := []trace.Hop{{TTL: 1, Addr: "10.0.0.1", Host: "gw", RTTs: []time.Duration{ms(2)}}}
		return trace.Result{Target: target, Addr: "203.0.113.5", Hops: h, Reached: true}, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	var updates []Update
	_ = Monitor(ctx, sweep, "example.com", false, 0, func(u Update) {
		updates = append(updates, u)
		if u.Round == 2 {
			cancel()
		}
	})
	if len(updates) != 2 {
		t.Fatalf("got %d updates, want 2", len(updates))
	}
	round2 := updates[1]
	if round2.Addr != "203.0.113.5" {
		t.Errorf("transient-error round dropped the dest addr: got %q, want 203.0.113.5", round2.Addr)
	}
	if !round2.Reached {
		t.Errorf("transient-error round dropped Reached")
	}
	// The lost sweep must NOT count as a sent probe.
	if len(round2.Hops) != 1 || round2.Hops[0].Sent != 1 {
		t.Errorf("lost sweep wrongly counted as a probe: hop Sent=%d, want 1", round2.Hops[0].Sent)
	}
}

func TestMonitorFirstSweepErrorReturns(t *testing.T) {
	boom := func(ctx context.Context, target string, useV6 bool, onHop func(trace.Hop)) (trace.Result, error) {
		return trace.Result{}, context.DeadlineExceeded
	}
	err := Monitor(context.Background(), boom, "bad.invalid", false, 0, func(Update) {})
	if err == nil {
		t.Fatal("first-sweep error should propagate, got nil")
	}
}
