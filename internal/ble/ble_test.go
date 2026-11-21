package ble

import (
	"testing"
	"time"
)

func TestAggregate_FoldsPacketsAndRanks(t *testing.T) {
	obs := []observation{
		// A far, unnamed device heard twice; strongest is -70.
		{address: "AA", rssi: -80},
		{address: "AA", rssi: -70},
		// A near device: name arrives on the 2nd packet, company on the 1st.
		{address: "BB", rssi: -55, company: 0x004C, hasCompany: true},
		{address: "BB", rssi: -50, name: "Keyboard"},
		// A mid device with an unknown company ID.
		{address: "CC", rssi: -60, name: "Sensor", company: 0x0157, hasCompany: true},
	}

	s := aggregate(obs, 6*time.Second)

	if s.DurationSec != 6 {
		t.Errorf("DurationSec = %d, want 6", s.DurationSec)
	}
	if s.Count != 3 {
		t.Errorf("Count = %d, want 3 distinct devices", s.Count)
	}
	if s.Named != 2 {
		t.Errorf("Named = %d, want 2", s.Named)
	}

	// Ranked strongest-first: BB (-50), CC (-60), AA (-70).
	wantOrder := []string{"BB", "CC", "AA"}
	for i, w := range wantOrder {
		if s.Devices[i].Address != w {
			t.Errorf("Devices[%d].Address = %s, want %s", i, s.Devices[i].Address, w)
		}
	}

	bb := s.Devices[0]
	if bb.RSSI != -50 {
		t.Errorf("BB.RSSI = %d, want -50 (strongest)", bb.RSSI)
	}
	if bb.Name != "Keyboard" {
		t.Errorf("BB.Name = %q, want Keyboard (backfilled from a later packet)", bb.Name)
	}
	if bb.Company != "Apple" {
		t.Errorf("BB.Company = %q, want Apple (0x004C)", bb.Company)
	}
	if bb.Seen != 2 {
		t.Errorf("BB.Seen = %d, want 2", bb.Seen)
	}

	cc := s.Devices[1]
	if cc.Company != "0x0157" {
		t.Errorf("CC.Company = %q, want the raw hex 0x0157 for an unknown ID", cc.Company)
	}

	aa := s.Devices[2]
	if aa.RSSI != -70 || aa.Name != "" || aa.Seen != 2 {
		t.Errorf("AA = {RSSI:%d Name:%q Seen:%d}, want {-70 \"\" 2}", aa.RSSI, aa.Name, aa.Seen)
	}
}

func TestAggregate_Empty(t *testing.T) {
	s := aggregate(nil, 3*time.Second)
	if s.Count != 0 || len(s.Devices) != 0 || s.Named != 0 {
		t.Errorf("empty survey = %+v, want zero devices", s)
	}
	if s.DurationSec != 3 {
		t.Errorf("DurationSec = %d, want 3", s.DurationSec)
	}
}

func TestCompanyLabel(t *testing.T) {
	if got := companyLabel(0x004C); got != "Apple" {
		t.Errorf("companyLabel(0x004C) = %q, want Apple", got)
	}
	if got := companyLabel(0x00E0); got != "Google" {
		t.Errorf("companyLabel(0x00E0) = %q, want Google", got)
	}
	if got := companyLabel(0x1234); got != "0x1234" {
		t.Errorf("companyLabel(0x1234) = %q, want raw hex 0x1234", got)
	}
}
