package configdiff

import "testing"

const cfgA = `hostname R1
!
interface GigabitEthernet0/1
 description uplink
 ip address 10.0.0.1 255.255.255.0
!
interface GigabitEthernet0/2
 shutdown
!`

const cfgB = `hostname R1
!
interface GigabitEthernet0/1
 description UPLINK to core
 ip address 10.0.0.1 255.255.255.0
!
interface GigabitEthernet0/3
 switchport mode access
!`

func TestDiffIdentical(t *testing.T) {
	r := Diff(cfgA, cfgA)
	if r.Added != 0 || r.Removed != 0 {
		t.Errorf("identical configs: want 0/0, got %d/%d", r.Added, r.Removed)
	}
	if len(r.Sections) != 0 {
		t.Errorf("identical configs: want no changed sections, got %v", r.Sections)
	}
	for _, l := range r.Lines {
		if l.Op != OpSame {
			t.Errorf("identical configs produced a %q line: %q", l.Op, l.Text)
		}
	}
}

func TestDiffChangedLineAttributedToSection(t *testing.T) {
	r := Diff(cfgA, cfgB)

	// the description edit: a del of the old line and an add of the new, both
	// under interface Gi0/1.
	var delOK, addOK bool
	for _, l := range r.Lines {
		if l.Op == OpDel && contains(l.Text, "description uplink") {
			if l.Section != "interface GigabitEthernet0/1" {
				t.Errorf("del line section = %q, want interface Gi0/1", l.Section)
			}
			delOK = true
		}
		if l.Op == OpAdd && contains(l.Text, "UPLINK to core") {
			if l.Section != "interface GigabitEthernet0/1" {
				t.Errorf("add line section = %q, want interface Gi0/1", l.Section)
			}
			addOK = true
		}
	}
	if !delOK || !addOK {
		t.Fatalf("description change not found (del=%v add=%v)", delOK, addOK)
	}
}

func TestDiffAddedAndRemovedSection(t *testing.T) {
	r := Diff(cfgA, cfgB)
	// Gi0/2 removed, Gi0/3 added.
	var removedG2, addedG3 bool
	for _, l := range r.Lines {
		if l.Op == OpDel && contains(l.Text, "GigabitEthernet0/2") {
			removedG2 = true
		}
		if l.Op == OpAdd && contains(l.Text, "GigabitEthernet0/3") {
			addedG3 = true
		}
	}
	if !removedG2 || !addedG3 {
		t.Errorf("want Gi0/2 removed and Gi0/3 added; got removed=%v added=%v", removedG2, addedG3)
	}
	if r.Added == 0 || r.Removed == 0 {
		t.Errorf("want nonzero added/removed counts, got %d/%d", r.Added, r.Removed)
	}
	// changed sections should include the three touched interfaces
	if !hasSection(r.Sections, "interface GigabitEthernet0/1") ||
		!hasSection(r.Sections, "interface GigabitEthernet0/2") ||
		!hasSection(r.Sections, "interface GigabitEthernet0/3") {
		t.Errorf("changed sections incomplete: %v", r.Sections)
	}
}

func TestDiffEmptyInputs(t *testing.T) {
	r := Diff("", "hostname R1")
	if r.Added != 1 {
		t.Errorf("want 1 added, got %d", r.Added)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func hasSection(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
