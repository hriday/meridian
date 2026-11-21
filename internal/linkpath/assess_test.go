package linkpath

import (
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/lldp"
)

func hasFinding(fs []Finding, level, sub string) bool {
	for _, f := range fs {
		if f.Level == level && strings.Contains(f.Text, sub) {
			return true
		}
	}
	return false
}

func TestAssess_switchFromLLDP(t *testing.T) {
	sw := lldp.Neighbor{Protocol: "LLDP", SystemName: "core-sw1", PortID: "Gi0/12", Capabilities: []string{"Bridge"}, MgmtIP: "192.168.1.2"}
	v, fs := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, []lldp.Neighbor{sw}, "ok", "192.168.1.1", mgmtProbe{}, "")
	if !strings.Contains(v, "switch is directly attached") {
		t.Errorf("verdict = %q, want a switch-detected verdict", v)
	}
	if !hasFinding(fs, "found", "core-sw1") || !hasFinding(fs, "found", "Gi0/12") {
		t.Errorf("expected a finding naming the switch and port: %+v", fs)
	}
	if !hasFinding(fs, "good", "Full-duplex") {
		t.Errorf("expected the full-duplex good finding")
	}
}

func TestAssess_hubFromHalfDuplex(t *testing.T) {
	v, fs := assess(linkState{known: true, duplex: "half", speedMbps: 100}, nil, "needs-privilege", "192.168.1.1", mgmtProbe{}, "")
	if !strings.Contains(v, "hub is almost certainly") {
		t.Errorf("verdict = %q, want a hub verdict", v)
	}
	if !hasFinding(fs, "found", "Half-duplex") {
		t.Errorf("expected the half-duplex finding: %+v", fs)
	}
	if !hasFinding(fs, "limit", "elevated privilege") {
		t.Errorf("expected the needs-privilege limit finding")
	}
}

func TestAssess_fullDuplexCleanButHedged(t *testing.T) {
	v, fs := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, nil, "ok", "192.168.1.1", mgmtProbe{}, "")
	if !strings.Contains(v, "No hub or announced switch") {
		t.Errorf("verdict = %q, want the clean-but-hedged verdict", v)
	}
	// The unmanaged-switch blind spot must always be stated.
	if !hasFinding(fs, "limit", "unmanaged switch is invisible") {
		t.Errorf("the unmanaged-switch caveat must always be present: %+v", fs)
	}
}

func TestAssess_fullDuplexNeedsPrivilegePointsToRecipe(t *testing.T) {
	v, _ := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, nil, "needs-privilege", "192.168.1.1", mgmtProbe{}, "")
	if !strings.Contains(v, "No hub detected") || !strings.Contains(v, "recipe") {
		t.Errorf("verdict = %q, want a 'no hub, run recipe' verdict", v)
	}
}

func TestAssess_routerItselfIsNotASwitch(t *testing.T) {
	gw := lldp.Neighbor{Protocol: "LLDP", SystemName: "gw", Capabilities: []string{"Router"}, MgmtIP: "192.168.1.1"}
	v, fs := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, []lldp.Neighbor{gw}, "ok", "192.168.1.1", mgmtProbe{}, "")
	if strings.Contains(v, "switch is directly attached") {
		t.Errorf("the gateway router must not be reported as an intermediate switch: %q", v)
	}
	if !hasFinding(fs, "info", "router itself") {
		t.Errorf("expected a finding identifying the device as the router: %+v", fs)
	}
}

func TestAssess_repeaterIsAHub(t *testing.T) {
	rep := lldp.Neighbor{Protocol: "LLDP", SystemName: "oldhub", Capabilities: []string{"Repeater"}}
	v, _ := assess(linkState{known: true, duplex: "full", speedMbps: 100}, []lldp.Neighbor{rep}, "ok", "192.168.1.1", mgmtProbe{}, "")
	if !strings.Contains(v, "hub is almost certainly") {
		t.Errorf("an LLDP Repeater should drive a hub verdict, got %q", v)
	}
}

func TestAssess_noneHeardWithPrivilegeIsNotToldToReRun(t *testing.T) {
	// LLDP ran (with privilege) and heard nothing — the verdict must NOT tell
	// the user to re-run with privilege they already supplied.
	v, _ := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, nil, "none-heard", "192.168.1.1", mgmtProbe{}, "")
	if strings.Contains(v, "needs privilege") || strings.Contains(v, "recipe") {
		t.Errorf("none-heard verdict shouldn't ask to re-run with privilege: %q", v)
	}
	if !strings.Contains(v, "No hub or announced switch") {
		t.Errorf("verdict = %q, want the clean-but-hedged verdict", v)
	}
}

func TestAssess_combinedRouterSwitchAtGatewayIsNotIntermediate(t *testing.T) {
	// A typical home router advertises both Bridge and Router at the gateway IP;
	// it's the router itself, not an intermediate switch.
	gw := lldp.Neighbor{Protocol: "LLDP", SystemName: "home-gw", Capabilities: []string{"Bridge", "Router"}, MgmtIP: "192.168.1.1"}
	v, fs := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, []lldp.Neighbor{gw}, "ok", "192.168.1.1", mgmtProbe{}, "")
	if strings.Contains(v, "switch is directly attached") {
		t.Errorf("a router+switch at the gateway IP must not read as an intermediate switch: %q", v)
	}
	if !hasFinding(fs, "info", "router itself") {
		t.Errorf("expected the gateway to be identified as the router: %+v", fs)
	}
}

func TestAssess_gigabitHalfDuplexIsIgnored(t *testing.T) {
	// A half-duplex reading at gigabit is physically impossible — don't let a
	// driver misreport produce a false hub verdict.
	v, _ := assess(linkState{known: true, duplex: "half", speedMbps: 1000}, nil, "ok", "192.168.1.1", mgmtProbe{}, "")
	if strings.Contains(v, "hub is almost certainly") {
		t.Errorf("a gigabit half-duplex misreport must not drive a hub verdict: %q", v)
	}
}

func TestAssess_gatewaySNMPFinding(t *testing.T) {
	// The unprivileged SNMP probe answered — surface it as a managed-device find.
	probe := mgmtProbe{snmpOK: true, sysDescr: "Cisco IOS Software, C3560 Switch", sysName: "core-sw1"}
	_, fs := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, nil, "needs-privilege", "192.168.1.1", probe, "hint")
	if !hasFinding(fs, "found", "answers SNMP") || !hasFinding(fs, "found", "Cisco IOS") {
		t.Errorf("expected an SNMP finding naming the gateway's sysDescr: %+v", fs)
	}
}

func TestAssess_needsPrivilegeIncludesSetupHint(t *testing.T) {
	_, fs := assess(linkState{known: true, duplex: "full", speedMbps: 1000}, nil, "needs-privilege", "192.168.1.1", mgmtProbe{}, "RUN-SETCAP-ONCE")
	if !hasFinding(fs, "limit", "RUN-SETCAP-ONCE") {
		t.Errorf("needs-privilege finding should include the one-time setup hint: %+v", fs)
	}
	if !hasFinding(fs, "limit", "snmpsweep") {
		t.Errorf("needs-privilege finding should point at the unprivileged fallback tools: %+v", fs)
	}
}

func TestAssess_wireless(t *testing.T) {
	v, fs := assess(linkState{wireless: true}, nil, "skipped-wireless", "192.168.1.1", mgmtProbe{}, "")
	if !strings.Contains(v, "Wi-Fi") {
		t.Errorf("verdict = %q, want the Wi-Fi verdict", v)
	}
	if !hasFinding(fs, "limit", "access point") {
		t.Errorf("expected the can't-see-past-AP caveat: %+v", fs)
	}
}
