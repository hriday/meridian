// Package netquality turns latency, jitter and packet loss into a real-time
// quality verdict — an estimated MOS (Mean Opinion Score, the standard 1–5 VoIP
// quality metric) via a simplified ITU-T G.107 E-model, plus a plain-English
// rating and what's limiting it. It answers "is this link good enough for a
// voice/video call?"
package netquality

import (
	"fmt"
	"math"
)

// Score is a connection-quality verdict.
type Score struct {
	MOS      float64 `json:"mos"`      // 1.0–4.5, estimated
	R        float64 `json:"r"`        // E-model R-factor, 0–100
	Rating   string  `json:"rating"`   // Excellent | Good | Fair | Poor | Bad
	Summary  string  `json:"summary"`  // one-line human verdict
	Limiting string  `json:"limiting"` // latency | jitter | loss | none
}

// Rate scores a link from its average latency, jitter (both ms) and loss (%).
func Rate(avgMs, jitterMs, lossPct float64) Score {
	// Effective latency folds in jitter (weighted) and codec/processing delay.
	eff := avgMs + 2*jitterMs + 10

	var r float64
	if eff < 160 {
		r = 93.2 - eff/40
	} else {
		r = 93.2 - (eff-120)/10
	}
	r -= 2.5 * lossPct // packet-loss impairment
	r = clamp(r, 0, 100)

	mos := 1 + 0.035*r + r*(r-60)*(100-r)*7e-6
	mos = clamp(mos, 1, 4.5)

	rating := ratingFor(mos)
	limiting := limitingFactor(avgMs, jitterMs, lossPct)
	return Score{
		MOS:      math.Round(mos*100) / 100,
		R:        math.Round(r*10) / 10,
		Rating:   rating,
		Limiting: limiting,
		Summary:  summaryFor(rating, limiting, avgMs, jitterMs, lossPct),
	}
}

func ratingFor(mos float64) string {
	switch {
	case mos >= 4.3:
		return "Excellent"
	case mos >= 4.0:
		return "Good"
	case mos >= 3.6:
		return "Fair"
	case mos >= 3.1:
		return "Poor"
	default:
		return "Bad"
	}
}

// limitingFactor names the metric contributing most, by how far each exceeds a
// "noticeable for real-time" threshold (loss 1%, jitter 30 ms, latency 150 ms).
func limitingFactor(avgMs, jitterMs, lossPct float64) string {
	worst, name := 0.0, "none"
	consider := func(sev float64, n string) {
		if sev > worst {
			worst, name = sev, n
		}
	}
	consider(lossPct/1.0, "loss")
	consider(jitterMs/30.0, "jitter")
	consider(avgMs/150.0, "latency")
	if worst < 1 {
		return "none"
	}
	return name
}

func summaryFor(rating, limiting string, avgMs, jitterMs, lossPct float64) string {
	if rating == "Excellent" || rating == "Good" {
		return rating + " for voice & video"
	}
	switch limiting {
	case "loss":
		return fmt.Sprintf("%s — %.1f%% packet loss is the main problem", rating, lossPct)
	case "jitter":
		return fmt.Sprintf("%s — jitter of %.0f ms is too high for real-time", rating, jitterMs)
	case "latency":
		return fmt.Sprintf("%s — %.0f ms latency is high for real-time", rating, avgMs)
	default:
		return rating + " for real-time traffic"
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
