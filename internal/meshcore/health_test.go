package meshcore

import (
	"strings"
	"testing"
)

func hasFinding(fs []Finding, level, substr string) bool {
	for _, f := range fs {
		if f.Level == level && strings.Contains(strings.ToLower(f.Title+" "+f.Fix), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

func TestDiagnose_NoContacts(t *testing.T) {
	s := Snapshot{
		Radio:    &RadioConfig{FreqMHz: 910.525, BandwidthKHz: 250, SpreadFactor: 11, CodingRate: 5},
		Contacts: []Contact{},
	}
	fs := diagnose(s)
	if !hasFinding(fs, "check", "knows no other nodes") {
		t.Errorf("expected an isolation finding, got %+v", fs)
	}
	// The advice must name the exact radio params a peer has to match.
	if !hasFinding(fs, "check", "910.525 mhz") || !hasFinding(fs, "check", "spreading factor 11") {
		t.Errorf("isolation advice should name the radio params, got %+v", fs)
	}
	// Frequency is set, so there must be no "no frequency" finding.
	if hasFinding(fs, "check", "no radio frequency") {
		t.Errorf("frequency is set; should not flag it: %+v", fs)
	}
}

func TestDiagnose_NoFrequency(t *testing.T) {
	s := Snapshot{Radio: &RadioConfig{FreqMHz: 0}, Contacts: []Contact{{Name: "x"}}}
	if !hasFinding(diagnose(s), "check", "no radio frequency") {
		t.Errorf("zero frequency should be flagged")
	}
}

func TestDiagnose_HealthyWithContacts(t *testing.T) {
	s := Snapshot{
		Radio:    &RadioConfig{FreqMHz: 910.525, BandwidthKHz: 250, SpreadFactor: 11, CodingRate: 5},
		Contacts: []Contact{{Name: "Hilltop"}},
	}
	if len(diagnose(s)) != 0 {
		t.Errorf("a node with contacts and a frequency should have no findings: %+v", diagnose(s))
	}
}

func TestRankContacts_MostRecentFirst(t *testing.T) {
	cs := []Contact{{Name: "old", LastSeen: 100}, {Name: "new", LastSeen: 200}}
	rankContacts(cs)
	if cs[0].Name != "new" {
		t.Errorf("most recently heard should rank first, got %+v", cs)
	}
}
