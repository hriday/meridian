package fwconfig

import (
	"strings"
	"testing"
)

// A realistic VyOS 1.3 (equuleus) set-format export: two named rule-sets whose
// rules match protocol/port/addresses, wired to zone pairs via zone-policy, with
// interfaces assigned to zones.
const vyosSample = `set firewall name LAN-DMZ default-action drop
set firewall name LAN-DMZ rule 10 action accept
set firewall name LAN-DMZ rule 10 protocol tcp
set firewall name LAN-DMZ rule 10 source address 10.1.0.0/16
set firewall name LAN-DMZ rule 10 destination address 10.2.10.0/24
set firewall name LAN-DMZ rule 10 destination port 443
set firewall name LAN-DMZ rule 20 action accept
set firewall name LAN-DMZ rule 20 protocol tcp
set firewall name LAN-DMZ rule 20 destination port 22
set firewall name DMZ-LAN default-action drop
set firewall name DMZ-LAN rule 10 action drop
set firewall name DMZ-LAN rule 10 protocol tcp
set firewall name DMZ-LAN rule 10 destination port 3389
set zone-policy zone LAN interface eth1
set zone-policy zone LAN default-action drop
set zone-policy zone DMZ interface eth2
set zone-policy zone DMZ default-action drop
set zone-policy zone DMZ from LAN firewall name LAN-DMZ
set zone-policy zone LAN from DMZ firewall name DMZ-LAN
`

func TestAnalyzeVyosEmpty(t *testing.T) {
	if _, err := analyzeVyos("  \n\t"); err == nil {
		t.Fatal("expected an error for empty config")
	}
}

func TestAnalyzeVyosCounts(t *testing.T) {
	as, err := analyzeVyos(vyosSample)
	if err != nil {
		t.Fatalf("analyzeVyos: %v", err)
	}
	if len(as) != 1 {
		t.Fatalf("want 1 analysis, got %d", len(as))
	}
	a := as[0]
	if a.Counts.Policies != 3 { // LAN-DMZ (2 rules) + DMZ-LAN (1 rule)
		t.Errorf("policies: want 3, got %d", a.Counts.Policies)
	}
	if a.Counts.Zones != 2 {
		t.Errorf("zones: want 2 (LAN, DMZ), got %d", a.Counts.Zones)
	}
}

func TestAnalyzeVyosZoneFlows(t *testing.T) {
	as, err := analyzeVyos(vyosSample)
	if err != nil {
		t.Fatalf("analyzeVyos: %v", err)
	}
	a := as[0]

	// LAN→DMZ ruleset is bound from zone LAN to zone DMZ; zones resolve to their
	// interfaces (eth1, eth2), like SRX. Both accept rules aggregate into one flow.
	allow := findFlow(a.Flows, "eth1", "eth2", "allow")
	if allow == nil {
		t.Fatalf("missing eth1 -> eth2 allow flow; flows=%+v", a.Flows)
	}
	if !hasService(allow.Services, "tcp/443") || !hasService(allow.Services, "tcp/22") {
		t.Errorf("LAN->DMZ allow flow services want tcp/443 and tcp/22; got %v", allow.Services)
	}

	// DMZ→LAN drops RDP.
	deny := findFlow(a.Flows, "eth2", "eth1", "deny")
	if deny == nil {
		t.Fatalf("missing eth2 -> eth1 deny flow; flows=%+v", a.Flows)
	}
	if !hasService(deny.Services, "tcp/3389") {
		t.Errorf("DMZ->LAN deny flow missing tcp/3389; got %v", deny.Services)
	}
}

// A rule-set can also be applied straight to an interface (no zones), which maps
// to interface → Internet, like ASA.
func TestAnalyzeVyosInterfaceBinding(t *testing.T) {
	cfg := `set firewall name WAN-IN rule 10 action accept
set firewall name WAN-IN rule 10 protocol tcp
set firewall name WAN-IN rule 10 destination port 443
set interfaces ethernet eth0 firewall in name WAN-IN
`
	as, err := analyzeVyos(cfg)
	if err != nil {
		t.Fatalf("analyzeVyos: %v", err)
	}
	a := as[0]
	if findFlow(a.Flows, "eth0", internetNode, "allow") == nil {
		t.Errorf("missing eth0 -> Internet allow flow; flows=%+v", a.Flows)
	}
}

// The 1.4 `firewall ipv4 name` spelling is accepted as an alias.
func TestAnalyzeVyosIpv4NameAlias(t *testing.T) {
	cfg := `set firewall ipv4 name R1 rule 10 action accept
set firewall ipv4 name R1 rule 10 protocol tcp
set firewall ipv4 name R1 rule 10 destination port 80
set interfaces ethernet eth0 firewall in name R1
`
	as, err := analyzeVyos(cfg)
	if err != nil {
		t.Fatalf("analyzeVyos: %v", err)
	}
	if as[0].Counts.Policies != 1 {
		t.Errorf("policies: want 1, got %d", as[0].Counts.Policies)
	}
}

// Multi-value matches use `group {network-group|address-group|port-group}`; those
// must survive so a port-restricted rule isn't read as "allow all tcp".
func TestAnalyzeVyosGroupReferences(t *testing.T) {
	cfg := `set firewall name WAN-LOCAL rule 20 action accept
set firewall name WAN-LOCAL rule 20 protocol tcp
set firewall name WAN-LOCAL rule 20 source group network-group INTERNAL
set firewall name WAN-LOCAL rule 20 destination group port-group MGMT-PORTS
set interfaces ethernet eth0 firewall in name WAN-LOCAL
`
	as, err := analyzeVyos(cfg)
	if err != nil {
		t.Fatalf("analyzeVyos: %v", err)
	}
	var p *Policy
	for i := range as[0].Policies {
		if as[0].Policies[i].Name == "WAN-LOCAL/20" {
			p = &as[0].Policies[i]
		}
	}
	if p == nil {
		t.Fatalf("policy WAN-LOCAL/20 not found")
	}
	if !hasService(p.Services, "tcp/MGMT-PORTS") {
		t.Errorf("destination port-group dropped; services=%v (want tcp/MGMT-PORTS)", p.Services)
	}
	var foundSrc bool
	for _, s := range p.SrcAddr {
		if strings.Contains(s, "INTERNAL") {
			foundSrc = true
		}
	}
	if !foundSrc {
		t.Errorf("source network-group dropped; srcAddr=%v", p.SrcAddr)
	}
}

// Pasting curly-brace `show configuration` (not `show configuration commands`)
// must be rejected, not silently return an empty analysis.
func TestAnalyzeVyosCurlyBraceRejected(t *testing.T) {
	curly := `firewall {
    name WAN-LOCAL {
        rule 20 {
            action accept
        }
    }
}`
	if _, err := analyzeVyos(curly); err == nil {
		t.Fatal("expected an error for curly-brace format (not set-format)")
	}
}
