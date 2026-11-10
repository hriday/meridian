package metrics

import "time"

// ThruSample is one poll of an interface's counters (plus optional CPU/mem).
type ThruSample struct {
	InOctets, OutOctets uint64
	InErrors, OutErrors uint64
	Bits                int // counter width: 64 (HC) or 32
	OperStatus          string
	CPUPct, MemPct      float64
	HasCPU, HasMem      bool
	T                   time.Time
}

// ThruSnapshot is the derived realtime view of an interface.
type ThruSnapshot struct {
	InBps       float64 `json:"inBps"`
	OutBps      float64 `json:"outBps"`
	UtilPct     float64 `json:"utilPct"`
	HasUtil     bool    `json:"hasUtil"`
	CPUPct      float64 `json:"cpuPct"`
	MemPct      float64 `json:"memPct"`
	HasCPU      bool    `json:"hasCpu"`
	HasMem      bool    `json:"hasMem"`
	InErrDelta  uint64  `json:"inErrDelta"`
	OutErrDelta uint64  `json:"outErrDelta"`
	OperStatus  string  `json:"operStatus"`
	BandLo      float64 `json:"bandLo"`
	BandHi      float64 `json:"bandHi"`
	State       string  `json:"state"` // warming | steady | busy | saturated | down | unreachable | error
	Warming     bool    `json:"warming"`
	Note        string  `json:"note"` // human message, e.g. for error/unreachable
}

// ThruAgg keeps the last `window` throughput samples for an interface and
// derives in/out bps, utilisation and a known-good band.
type ThruAgg struct {
	window   int
	warmupN  int
	speedBps uint64
	prev     *ThruSample
	bps      []float64 // total (in+out) bps history
	frozen   bool
	frozenLo float64
	frozenHi float64
}

const thruWarmup = 5

// NewThruAgg returns an aggregator over the last `window` samples for a link of
// speedBps bits/s (0 when unknown — utilisation is then "n/a").
func NewThruAgg(window int, speedBps uint64) *ThruAgg {
	if window < thruWarmup {
		window = thruWarmup
	}
	return &ThruAgg{window: window, warmupN: thruWarmup, speedBps: speedBps}
}

// SetBaseline freezes the current throughput band as the known-good range.
func (a *ThruAgg) SetBaseline() {
	a.frozenLo, a.frozenHi = a.computeBand()
	a.frozen = true
}

// Add records a sample and returns the current snapshot.
func (a *ThruAgg) Add(s ThruSample) ThruSnapshot {
	snap := ThruSnapshot{
		OperStatus: s.OperStatus,
		CPUPct:     s.CPUPct, MemPct: s.MemPct, HasCPU: s.HasCPU, HasMem: s.HasMem,
	}
	if a.prev != nil {
		if dt := s.T.Sub(a.prev.T).Seconds(); dt > 0 {
			inD, inOK := counterDelta(s.InOctets, a.prev.InOctets, s.Bits)
			outD, outOK := counterDelta(s.OutOctets, a.prev.OutOctets, s.Bits)
			if inOK {
				snap.InBps = float64(inD) * 8 / dt
			}
			if outOK {
				snap.OutBps = float64(outD) * 8 / dt
			}
			if s.InErrors >= a.prev.InErrors {
				snap.InErrDelta = s.InErrors - a.prev.InErrors
			}
			if s.OutErrors >= a.prev.OutErrors {
				snap.OutErrDelta = s.OutErrors - a.prev.OutErrors
			}
			if inOK && outOK {
				a.bps = append(a.bps, snap.InBps+snap.OutBps)
				if len(a.bps) > a.window {
					a.bps = a.bps[len(a.bps)-a.window:]
				}
			}
		}
	}
	cp := s
	a.prev = &cp

	if a.speedBps > 0 {
		snap.HasUtil = true
		m := snap.InBps
		if snap.OutBps > m {
			m = snap.OutBps
		}
		snap.UtilPct = m / float64(a.speedBps) * 100
	}

	lo, hi := a.computeBand()
	if a.frozen {
		lo, hi = a.frozenLo, a.frozenHi
	}
	snap.BandLo, snap.BandHi = lo, hi

	snap.Warming = len(a.bps) < a.warmupN
	switch {
	case s.OperStatus != "" && s.OperStatus != "up":
		snap.State = "down"
	case snap.Warming:
		snap.State = "warming"
	case snap.HasUtil && snap.UtilPct > 90:
		snap.State = "saturated"
	case snap.HasUtil && snap.UtilPct > 70:
		snap.State = "busy"
	default:
		snap.State = "steady"
	}
	return snap
}

func (a *ThruAgg) computeBand() (float64, float64) {
	if len(a.bps) == 0 {
		return 0, 0
	}
	med := median(a.bps)
	spread := madK * medianAbsDev(a.bps, med)
	if floor := med*0.15 + 1_000_000; spread < floor { // ≥ ~1 Mbps band
		spread = floor
	}
	lo := med - spread
	if lo < 0 {
		lo = 0
	}
	return lo, med + spread
}

// counterDelta returns the increase from prev to cur for a `bits`-wide counter,
// treating a decrease as a wrap only when prev was near the counter's maximum
// (otherwise it's a reset and ok is false → drop the sample).
func counterDelta(cur, prev uint64, bits int) (uint64, bool) {
	if cur >= prev {
		return cur - prev, true
	}
	if bits == 32 {
		const mod = 1 << 32
		const max = mod - 1
		if float64(prev) > 0.9*float64(max) {
			return (mod - prev) + cur, true
		}
		return 0, false
	}
	// 64-bit: near-max wrap; uint64 subtraction yields the correct wrapped delta.
	if float64(prev) > 0.9*1.8446744073709552e19 {
		return cur - prev, true
	}
	return 0, false
}
