package snmp

import "testing"

func TestSwitchPortsJoinsTables(t *testing.T) {
	macBytes := string([]byte{0x00, 0x0c, 0x29, 0x38, 0x78, 0x90}) // 00:0c:29:38:78:90
	f := &fakeClient{walk: map[string][]PDU{
		oidFdbPort:    {{OID: oidFdbPort + ".0.12.41.56.120.144", Value: 3}},
		oidBasePortIf: {{OID: oidBasePortIf + ".3", Value: 10001}},
		oidIfName:     {{OID: oidIfName + ".10001", Value: "GigabitEthernet0/1"}},
		oidArpPhys:    {{OID: oidArpPhys + ".500.10.1.1.5", Value: macBytes}},
	}}
	ents, err := SwitchPorts(f)
	if err != nil {
		t.Fatalf("SwitchPorts: %v", err)
	}
	if len(ents) != 1 {
		t.Fatalf("want 1 entry, got %d: %+v", len(ents), ents)
	}
	e := ents[0]
	if e.MAC != "00:0c:29:38:78:90" {
		t.Errorf("mac = %q, want 00:0c:29:38:78:90", e.MAC)
	}
	if e.Port != "GigabitEthernet0/1" {
		t.Errorf("port = %q, want GigabitEthernet0/1", e.Port)
	}
	if e.IP != "10.1.1.5" {
		t.Errorf("ip = %q, want 10.1.1.5", e.IP)
	}
	if e.IfIndex != 10001 {
		t.Errorf("ifIndex = %d, want 10001", e.IfIndex)
	}
}

// With no ARP data (or no name), an entry still maps MAC → port, just without an
// IP, and falls back to an ifN label.
func TestSwitchPortsNoArpOrName(t *testing.T) {
	f := &fakeClient{walk: map[string][]PDU{
		oidFdbPort:    {{OID: oidFdbPort + ".170.187.204.221.238.255", Value: 7}}, // aa:bb:cc:dd:ee:ff
		oidBasePortIf: {{OID: oidBasePortIf + ".7", Value: 42}},
	}}
	ents, err := SwitchPorts(f)
	if err != nil {
		t.Fatalf("SwitchPorts: %v", err)
	}
	if len(ents) != 1 {
		t.Fatalf("want 1 entry, got %d", len(ents))
	}
	if ents[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("mac = %q", ents[0].MAC)
	}
	if ents[0].IP != "" {
		t.Errorf("ip should be empty, got %q", ents[0].IP)
	}
	if ents[0].Port != "if42" {
		t.Errorf("port = %q, want if42 fallback", ents[0].Port)
	}
}

// A fabricated error from the required FDB walk propagates.
func TestSwitchPortsFDBError(t *testing.T) {
	if _, err := SwitchPorts(errClient{}); err == nil {
		t.Error("expected the FDB walk error to propagate")
	}
}

type errClient struct{}

func (errClient) Get([]string) ([]PDU, error) { return nil, nil }
func (errClient) Walk(string) ([]PDU, error)  { return nil, errFake }
func (errClient) Close() error                { return nil }

var errFake = fmtErr("snmp down")

type fmtErr string

func (e fmtErr) Error() string { return string(e) }
