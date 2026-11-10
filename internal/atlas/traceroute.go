package atlas

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Hop is one hop on a probe's traceroute path.
type Hop struct {
	Number  int     `json:"number"`
	From    string  `json:"from"`
	RttMs   float64 `json:"rttMs"`
	Timeout bool    `json:"timeout"`
}

// TraceProbe is one probe's traceroute to the target.
type TraceProbe struct {
	ID      int    `json:"id"`
	From    string `json:"from"`
	Dest    string `json:"dest"`
	Hops    []Hop  `json:"hops"`
	Reached bool   `json:"reached"`
}

// TraceReport aggregates a traceroute measurement across probes.
type TraceReport struct {
	Target        string       `json:"target"`
	MeasurementID int          `json:"measurementId"`
	Probes        []TraceProbe `json:"probes"`
	Responding    int          `json:"responding"`
	Reached       int          `json:"reached"`
}

// scheduleTrace POSTs a one-off ICMP traceroute measurement and returns its id.
func scheduleTrace(ctx context.Context, d Doer, key, target string, probes int) (int, error) {
	return scheduleMeasurement(ctx, d, key, map[string]any{
		"type":        "traceroute",
		"af":          afFor(target),
		"target":      target,
		"description": "Meridian remote traceroute " + target,
		"protocol":    "ICMP",
		"is_oneoff":   true,
	}, probes)
}

// rawTraceReply is one probe/packet reply at a hop; a timeout has x == "*".
type rawTraceReply struct {
	From string  `json:"from"`
	Rtt  float64 `json:"rtt"`
	X    string  `json:"x"`
}

type rawTraceHop struct {
	Hop    int             `json:"hop"`
	Result []rawTraceReply `json:"result"`
}

type rawTrace struct {
	PrbID   int           `json:"prb_id"`
	From    string        `json:"from"`
	DstAddr string        `json:"dst_addr"`
	Result  []rawTraceHop `json:"result"`
}

// traceResults fetches and parses the per-probe traceroute results.
func traceResults(ctx context.Context, d Doer, key string, id int) ([]TraceProbe, error) {
	data, err := fetchResults(ctx, d, key, id)
	if err != nil {
		return nil, err
	}
	var raw []rawTrace
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("atlas: unexpected traceroute results: %w", err)
	}

	probes := make([]TraceProbe, 0, len(raw))
	for _, rt := range raw {
		p := TraceProbe{ID: rt.PrbID, From: rt.From, Dest: rt.DstAddr}
		for _, rh := range rt.Result {
			hop := Hop{Number: rh.Hop, Timeout: true}
			for _, reply := range rh.Result {
				if reply.From == "" {
					continue // timeout / non-responding packet
				}
				// first responder sets the address; keep the minimum RTT.
				if hop.Timeout || reply.Rtt < hop.RttMs {
					hop.RttMs = reply.Rtt
				}
				if hop.From == "" {
					hop.From = reply.From
				}
				hop.Timeout = false
				if reply.From == rt.DstAddr && rt.DstAddr != "" {
					p.Reached = true
				}
			}
			p.Hops = append(p.Hops, hop)
		}
		probes = append(probes, p)
	}
	return probes, nil
}

// Trace schedules a one-off traceroute from the given number of worldwide
// probes, polls until some report or the timeout elapses, and aggregates. A
// timeout with nothing reported yields an empty report and no error.
func Trace(ctx context.Context, d Doer, key, target string, probes int, poll, timeout time.Duration) (TraceReport, error) {
	id, err := scheduleTrace(ctx, d, key, target, probes)
	if err != nil {
		return TraceReport{}, err
	}
	rep := TraceReport{Target: target, MeasurementID: id}

	deadline := time.Now().Add(timeout)
	for {
		ps, err := traceResults(ctx, d, key, id)
		if err != nil {
			if ctx.Err() != nil {
				return rep, nil
			}
			return TraceReport{}, err
		}
		if len(ps) > 0 {
			rep.Probes = ps
			aggregateTrace(&rep)
			return rep, nil
		}
		if !time.Now().Before(deadline) {
			return rep, nil
		}
		select {
		case <-ctx.Done():
			return rep, nil
		case <-time.After(poll):
		}
	}
}

// aggregateTrace fills Responding (probes that produced any hops) and Reached.
func aggregateTrace(rep *TraceReport) {
	for _, p := range rep.Probes {
		if len(p.Hops) > 0 {
			rep.Responding++
		}
		if p.Reached {
			rep.Reached++
		}
	}
}
