// Package speedtest is Meridian's LAN throughput test: a thin, client-only
// wrapper over the pure-Go iperf3 client (github.com/appshubcc/go-iperf3). It
// runs a download/upload/bidirectional test against a host running `iperf3 -s`,
// streaming live samples. Meridian never serves — the library's server is used
// only in this package's tests (loopback).
package speedtest

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	iperf3 "github.com/appshubcc/go-iperf3"
)

// Options configures one test.
type Options struct {
	Host      string
	Port      int
	Direction string // download | upload | bidir
	Parallel  int
	Duration  time.Duration
	UDP       bool
}

// Sample is one live update while the test runs.
type Sample struct {
	Elapsed  float64 `json:"elapsed"`
	DownMbps float64 `json:"downMbps"`
	UpMbps   float64 `json:"upMbps"`
	Phase    string  `json:"phase"` // connecting | running
}

// Result is the final outcome.
type Result struct {
	DownMbps    float64 `json:"downMbps"`
	UpMbps      float64 `json:"upMbps"`
	Bytes       float64 `json:"bytes"`
	Retransmits int64   `json:"retransmits"`
	RTTMeanMs   float64 `json:"rttMeanMs"`
	JitterMs    float64 `json:"jitterMs"`
	Lost        int64   `json:"lost"`
	Packets     int64   `json:"packets"`
	Peer        string  `json:"peer"`
	Protocol    string  `json:"protocol"`
}

// Run executes the test, forwarding live (connecting/running) updates to
// onSample and returning the final Result. The terminal outcome is the returned
// Result/error — never a sample. ctx cancels the test.
func Run(ctx context.Context, o Options, onSample func(Sample)) (Result, error) {
	host := strings.TrimSpace(o.Host)
	if host == "" {
		return Result{}, fmt.Errorf("enter a host running iperf3 -s")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err // already cancelled — not a resolve failure
	}
	// Pre-resolve for a clear DNS error — the library has no DNS error code.
	rctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	_, rerr := net.DefaultResolver.LookupHost(rctx, host)
	cancel()
	if rerr != nil {
		if cerr := ctx.Err(); cerr != nil {
			return Result{}, cerr
		}
		return Result{}, fmt.Errorf("couldn't resolve %q", host)
	}

	dir := iperf3.Upload
	switch strings.ToLower(o.Direction) {
	case "download":
		dir = iperf3.Download
	case "bidir", "both":
		dir = iperf3.Bidir
	}
	proto := iperf3.TCP
	if o.UDP {
		proto = iperf3.UDP
	}

	st := iperf3.RunClient(ctx, iperf3.ClientConfig{
		Hosts:     []string{host},
		Port:      o.Port,
		Protocol:  proto,
		Direction: dir,
		Parallel:  o.Parallel,
		Duration:  o.Duration,
		OnUpdate: func(s iperf3.State) {
			if s.Phase != iperf3.PhaseConnecting && s.Phase != iperf3.PhaseRunning {
				return // terminal state is surfaced via the returned Result/error
			}
			onSample(Sample{
				Elapsed:  s.Elapsed,
				DownMbps: s.Download / 1e6,
				UpMbps:   s.Upload / 1e6,
				Phase:    string(s.Phase),
			})
		},
	})
	if st.Error != "" {
		return Result{}, mapError(st.Error, host, o.Port)
	}
	return buildResult(st), nil
}

// mapError turns an iperf3 State.Error code into a friendly error.
func mapError(code, host string, port int) error {
	if port == 0 {
		port = iperf3.DefaultPort
	}
	switch code {
	case "refused", "timeout", "unreachable":
		return fmt.Errorf("no iperf3 server answered at %s:%d — start `iperf3 -s` there", host, port)
	case "busy":
		return fmt.Errorf("that iperf3 server is busy running another test — try again shortly")
	case "cancelled":
		return context.Canceled // our own Stop; the app's ctx.Err guard suppresses it
	case "interrupted":
		return fmt.Errorf("the connection to %s dropped mid-test — the server may have stopped or the link went down", host)
	case "protocol":
		return fmt.Errorf("%s:%d answered but isn't an iperf3 server — check the host and port", host, port)
	case "udp":
		return fmt.Errorf("the UDP test couldn't be set up — the server may not accept UDP")
	default:
		if strings.HasPrefix(code, "server:") {
			return fmt.Errorf("the iperf3 server reported an error (%s)", code)
		}
		return fmt.Errorf("the speed test failed (%s)", code)
	}
}

// buildResult assembles the Result from the terminal State. Both result flows
// are set for every direction (an inactive direction has ~0 bitrate); the scalar
// extras come from the active flow (download preferred), bytes are the total.
func buildResult(st iperf3.State) Result {
	r := Result{Peer: st.Peer, Protocol: string(st.Protocol)}
	var primary *iperf3.Flow
	if st.DownloadResult != nil {
		r.DownMbps = st.DownloadResult.Bitrate / 1e6
		r.Bytes += st.DownloadResult.Bytes
		if st.DownloadResult.Bitrate > 0 {
			primary = st.DownloadResult
		}
	}
	if st.UploadResult != nil {
		r.UpMbps = st.UploadResult.Bitrate / 1e6
		r.Bytes += st.UploadResult.Bytes
		if primary == nil && st.UploadResult.Bitrate > 0 {
			primary = st.UploadResult
		}
	}
	if primary != nil {
		r.Retransmits = primary.Retransmits
		r.RTTMeanMs = primary.RTTMean
		r.JitterMs = primary.Jitter
		r.Lost = primary.Lost
		r.Packets = primary.Packets
	}
	return r
}
