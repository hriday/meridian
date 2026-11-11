package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/speedtest"
)

// maxSpeedtestSeconds caps a test so a long-running test can't be started by
// accident; the UI also caps it.
const maxSpeedtestSeconds = 60

// SpeedSnapshot is one streamed LAN speed-test update on "metrics:<streamID>".
type SpeedSnapshot struct {
	Elapsed  float64 `json:"elapsed"`
	DownMbps float64 `json:"downMbps"`
	UpMbps   float64 `json:"upMbps"`
	State    string  `json:"state"` // running | done | error
	Note     string  `json:"note"`
	Peer     string  `json:"peer"`
	Protocol string  `json:"protocol"`
	// set on done:
	AvgDownMbps float64 `json:"avgDownMbps"`
	AvgUpMbps   float64 `json:"avgUpMbps"`
	Bytes       float64 `json:"bytes"`
	Retransmits int64   `json:"retransmits"`
	RTTMeanMs   float64 `json:"rttMeanMs"`
	JitterMs    float64 `json:"jitterMs"`
	Lost        int64   `json:"lost"`
	Packets     int64   `json:"packets"`
}

// SpeedtestStart runs a LAN throughput test against host (an `iperf3 -s`),
// streaming a SpeedSnapshot per update on "metrics:<streamID>". The frontend
// generates the id and subscribes before calling this, so no sample is missed.
func (a *App) SpeedtestStart(streamID, host string, port int, direction string, parallel, durationSec int, udp bool) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("enter a host running iperf3 -s")
	}
	if durationSec <= 0 {
		durationSec = 10
	}
	if durationSec > maxSpeedtestSeconds {
		durationSec = maxSpeedtestSeconds
	}
	if parallel <= 0 {
		parallel = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &monitor{cancel: cancel, done: make(chan struct{})}
	if err := a.registerMonitor(streamID, m); err != nil {
		cancel()
		return err
	}
	event := "metrics:" + streamID
	emitCtx := a.lookupCtx()
	opts := speedtest.Options{
		Host: host, Port: port, Direction: direction, Parallel: parallel,
		Duration: time.Duration(durationSec) * time.Second, UDP: udp,
	}

	go func() {
		defer close(m.done)
		defer func() {
			if r := recover(); r != nil && ctx.Err() == nil {
				emitEvent(emitCtx, event, SpeedSnapshot{State: "error", Note: fmt.Sprintf("the speed test crashed: %v", r)})
			}
		}()
		res, err := speedtest.Run(ctx, opts, func(s speedtest.Sample) {
			emitEvent(emitCtx, event, SpeedSnapshot{
				Elapsed: s.Elapsed, DownMbps: s.DownMbps, UpMbps: s.UpMbps, State: "running",
			})
		})
		if ctx.Err() != nil {
			return // stopped — emit no terminal event
		}
		if err != nil {
			emitEvent(emitCtx, event, SpeedSnapshot{State: "error", Note: err.Error()})
			return
		}
		emitEvent(emitCtx, event, SpeedSnapshot{
			State: "done", Peer: res.Peer, Protocol: res.Protocol,
			AvgDownMbps: res.DownMbps, AvgUpMbps: res.UpMbps, Bytes: res.Bytes,
			Retransmits: res.Retransmits, RTTMeanMs: res.RTTMeanMs,
			JitterMs: res.JitterMs, Lost: res.Lost, Packets: res.Packets,
		})
	}()
	return nil
}

// SpeedtestStop cancels a running test and blocks until its goroutine exits.
func (a *App) SpeedtestStop(streamID string) { a.stopMonitor(streamID) }
