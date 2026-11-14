package netquality

import "testing"

func TestRateExcellent(t *testing.T) {
	s := Rate(10, 1, 0) // low latency, negligible jitter, no loss
	if s.Rating != "Excellent" {
		t.Errorf("rating = %q, want Excellent (MOS %.2f)", s.Rating, s.MOS)
	}
	if s.MOS < 4.3 {
		t.Errorf("MOS = %.2f, want >= 4.3", s.MOS)
	}
	if s.Limiting != "none" {
		t.Errorf("limiting = %q, want none", s.Limiting)
	}
}

func TestRateBadWithLoss(t *testing.T) {
	s := Rate(300, 100, 25) // terrible all round
	if s.Rating != "Bad" {
		t.Errorf("rating = %q, want Bad (MOS %.2f)", s.Rating, s.MOS)
	}
	if s.MOS > 2.0 {
		t.Errorf("MOS = %.2f, want low", s.MOS)
	}
	if s.Limiting != "loss" {
		t.Errorf("limiting = %q, want loss", s.Limiting)
	}
}

func TestLimitingIsJitter(t *testing.T) {
	s := Rate(20, 90, 0) // fine latency, no loss, huge jitter
	if s.Limiting != "jitter" {
		t.Errorf("limiting = %q, want jitter", s.Limiting)
	}
}

func TestLimitingIsLatency(t *testing.T) {
	s := Rate(400, 5, 0) // high latency, low jitter, no loss
	if s.Limiting != "latency" {
		t.Errorf("limiting = %q, want latency", s.Limiting)
	}
}

func TestMonotonic(t *testing.T) {
	// More loss must never improve the score.
	a := Rate(30, 10, 0)
	b := Rate(30, 10, 5)
	if b.MOS > a.MOS {
		t.Errorf("more loss raised MOS: %.2f -> %.2f", a.MOS, b.MOS)
	}
}
