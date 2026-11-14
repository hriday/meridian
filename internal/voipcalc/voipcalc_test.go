package voipcalc

import "testing"

func TestG711Known(t *testing.T) {
	// G.711 @ 20ms: 160B payload, +40B IP/UDP/RTP = 200B, @50pps => 80 kbps one-way.
	r, err := Calc("G.711", 20, 1)
	if err != nil {
		t.Fatalf("Calc: %v", err)
	}
	if r.PayloadBytes != 160 {
		t.Errorf("payload = %d, want 160", r.PayloadBytes)
	}
	if r.OneWayKbps != 80 {
		t.Errorf("one-way = %.1f, want 80", r.OneWayKbps)
	}
	if r.OneWayEthKbps != 87.2 { // (160+40+18)*8/20
		t.Errorf("one-way+eth = %.1f, want 87.2", r.OneWayEthKbps)
	}
	if r.PerCallKbps != 160 {
		t.Errorf("per-call = %.1f, want 160 (both directions)", r.PerCallKbps)
	}
}

func TestG729AndCalls(t *testing.T) {
	// G.729 @ 20ms: 20B payload, +40 = 60B, 50pps => 24 kbps one-way.
	r, err := Calc("G.729", 20, 10)
	if err != nil {
		t.Fatal(err)
	}
	if r.PayloadBytes != 20 || r.OneWayKbps != 24 {
		t.Errorf("G.729: payload=%d oneWay=%.1f, want 20 / 24", r.PayloadBytes, r.OneWayKbps)
	}
	if r.Calls != 10 || r.TotalKbps != 24*2*10 {
		t.Errorf("10 calls total = %.1f, want %d", r.TotalKbps, 24*2*10)
	}
}

func TestPtimeDefaultAndOverride(t *testing.T) {
	def, _ := Calc("G.711", 0, 1) // default 20ms
	if def.PtimeMs != 20 {
		t.Errorf("default ptime = %d, want 20", def.PtimeMs)
	}
	// Larger ptime = fewer packets = less overhead = lower bandwidth.
	big, _ := Calc("G.711", 40, 1)
	if big.OneWayKbps >= def.OneWayKbps {
		t.Errorf("40ms should be cheaper than 20ms: %.1f vs %.1f", big.OneWayKbps, def.OneWayKbps)
	}
}

func TestUnknownCodec(t *testing.T) {
	if _, err := Calc("MP3", 20, 1); err == nil {
		t.Error("unknown codec should error")
	}
}
