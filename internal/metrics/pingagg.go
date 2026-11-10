// Package metrics holds the pure, offline aggregation behind Meridian's
// realtime monitors: samples in, a Snapshot out (rolling stats + a known-good
// band + an at-a-glance state). No I/O, no Wails.
package metrics

import "sort"

// Snapshot is the current state of a ping monitor's rolling window.
type Snapshot struct {
	RTTMs    float64 `json:"rttMs"` // latest responding RTT (0 when the latest probe was lost)
	AvgMs    float64 `json:"avgMs"`
	LossPct  float64 `json:"lossPct"`
	JitterMs float64 `json:"jitterMs"`
	BandLo   float64 `json:"bandLo"`
	BandHi   float64 `json:"bandHi"`
	State    string  `json:"state"` // warming | steady | degraded | loss | anomaly
	Warming  bool    `json:"warming"`
	HasRTT   bool    `json:"hasRtt"`
}

type sample struct {
	rtt float64
	ok  bool
}

// Aggregator keeps the last `window` ping samples and derives a Snapshot.
type Aggregator struct {
	window   int
	warmupN  int
	samples  []sample
	frozen   bool
	frozenLo float64
	frozenHi float64
}

const (
	defaultWarmup = 10
	lossHighPct   = 10.0 // loss% at or above this (or a lost latest probe) => "loss"
	madK          = 3.0
)

// NewAggregator returns an aggregator over the last `window` samples.
func NewAggregator(window int) *Aggregator {
	if window < defaultWarmup {
		window = defaultWarmup
	}
	return &Aggregator{window: window, warmupN: defaultWarmup}
}

// Add records a sample (ok=false means the probe was lost; rtt ignored) and
// returns the current Snapshot.
func (a *Aggregator) Add(rttMs float64, ok bool) Snapshot {
	a.samples = append(a.samples, sample{rttMs, ok})
	if len(a.samples) > a.window {
		a.samples = a.samples[len(a.samples)-a.window:]
	}
	return a.snapshot()
}

// SetBaseline freezes the current band as the known-good range until reset.
func (a *Aggregator) SetBaseline() {
	a.frozenLo, a.frozenHi = a.computeBand()
	a.frozen = true
}

func (a *Aggregator) snapshot() Snapshot {
	n := len(a.samples)
	latest := a.samples[n-1]

	var rtts []float64
	lost := 0
	for _, s := range a.samples {
		if s.ok {
			rtts = append(rtts, s.rtt)
		} else {
			lost++
		}
	}
	lossPct := float64(lost) / float64(n) * 100

	lo, hi := a.computeBand()
	if a.frozen {
		lo, hi = a.frozenLo, a.frozenHi
	}

	s := Snapshot{
		AvgMs:    mean(rtts),
		LossPct:  lossPct,
		JitterMs: meanAbsDiff(rtts),
		BandLo:   lo,
		BandHi:   hi,
		HasRTT:   latest.ok,
		Warming:  n < a.warmupN,
	}
	if latest.ok {
		s.RTTMs = latest.rtt
	}

	switch {
	case s.Warming:
		s.State = "warming"
	case !latest.ok || lossPct >= lossHighPct:
		s.State = "loss"
	case lossPct > 0:
		s.State = "degraded"
	case latest.ok && (latest.rtt < lo || latest.rtt > hi):
		s.State = "anomaly"
	default:
		s.State = "steady"
	}
	return s
}

// computeBand is a robust known-good range over the responding RTTs: median ±
// k·MAD, with a floor so a dead-steady series still has a non-zero band.
func (a *Aggregator) computeBand() (float64, float64) {
	var rtts []float64
	for _, s := range a.samples {
		if s.ok {
			rtts = append(rtts, s.rtt)
		}
	}
	if len(rtts) == 0 {
		return 0, 0
	}
	med := median(rtts)
	spread := madK * medianAbsDev(rtts, med)
	if floor := med*0.15 + 1; spread < floor {
		spread = floor
	}
	lo := med - spread
	if lo < 0 {
		lo = 0
	}
	return lo, med + spread
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func meanAbsDiff(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var s float64
	for i := 1; i < len(xs); i++ {
		d := xs[i] - xs[i-1]
		if d < 0 {
			d = -d
		}
		s += d
	}
	return s / float64(len(xs)-1)
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	c := append([]float64(nil), xs...)
	sort.Float64s(c)
	n := len(c)
	if n%2 == 1 {
		return c[n/2]
	}
	return (c[n/2-1] + c[n/2]) / 2
}

func medianAbsDev(xs []float64, med float64) float64 {
	dev := make([]float64, len(xs))
	for i, x := range xs {
		d := x - med
		if d < 0 {
			d = -d
		}
		dev[i] = d
	}
	return median(dev)
}
