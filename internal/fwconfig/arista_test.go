package fwconfig

import "testing"

// A realistic Arista EOS `show running-config` excerpt: named ACLs with
// sequence-numbered rules (CIDR / host / any), a standard ACL that must be
// ignored, and interfaces that bind an ACL with `ip access-group ... in`.
const aristaSample = `!
ip access-list WEB_IN
   10 permit tcp 203.0.113.0/24 host 10.1.10.10 eq https
   20 permit tcp any host 10.1.10.10 eq http
   30 deny ip any any
!
ip access-list standard MGMT
   10 permit host 10.0.0.5
!
ip access-list DB_IN
   10 permit tcp 10.1.10.0/24 10.1.20.0/24 eq 5432
   20 deny ip any any
!
interface Ethernet1
   description Uplink to ISP
   no switchport
   ip address 203.0.113.2/30
   ip access-group WEB_IN in
!
interface Vlan10
   ip address 10.1.10.1/24
!
interface Vlan20
   ip address 10.1.20.1/24
   ip access-group DB_IN in
!
`

func TestAnalyzeAristaEmpty(t *testing.T) {
	if _, err := analyzeArista("   \n\t"); err == nil {
		t.Fatal("expected an error for empty config")
	}
}

func TestAnalyzeAristaCounts(t *testing.T) {
	as, err := analyzeArista(aristaSample)
	if err != nil {
		t.Fatalf("analyzeArista: %v", err)
	}
	if len(as) != 1 {
		t.Fatalf("want 1 analysis, got %d", len(as))
	}
	a := as[0]
	// WEB_IN (3) + DB_IN (2); the standard ACL MGMT contributes none.
	if a.Counts.Policies != 5 {
		t.Errorf("policies: want 5, got %d", a.Counts.Policies)
	}
	if a.Counts.Zones != 3 {
		t.Errorf("zones: want 3, got %d", a.Counts.Zones)
	}
	// Ethernet1 /30, Vlan10 /24, Vlan20 /24 — three distinct subnets.
	if a.Counts.SubnetsIPs != 3 {
		t.Errorf("subnetsIps: want 3, got %d", a.Counts.SubnetsIPs)
	}
}

func TestAnalyzeAristaStandardAclIgnored(t *testing.T) {
	as, _ := analyzeArista(aristaSample)
	for _, p := range as[0].Policies {
		if p.Name == "MGMT" {
			t.Fatalf("standard ACL MGMT must not produce a policy")
		}
	}
}

func TestAnalyzeAristaFlows(t *testing.T) {
	as, err := analyzeArista(aristaSample)
	if err != nil {
		t.Fatalf("analyzeArista: %v", err)
	}
	a := as[0]

	// WEB_IN is bound `in` on Ethernet1 → Ethernet1 → Internet (any), allow,
	// carrying the two permitted services.
	web := findFlow(a.Flows, "Ethernet1", internetNode, "allow")
	if web == nil {
		t.Fatalf("missing Ethernet1 -> Internet allow flow; flows=%+v", a.Flows)
	}
	if !hasService(web.Services, "tcp/https") {
		t.Errorf("Ethernet1 allow flow missing tcp/https; got %v", web.Services)
	}

	// DB_IN is bound `in` on Vlan20.
	db := findFlow(a.Flows, "Vlan20", internetNode, "allow")
	if db == nil {
		t.Fatalf("missing Vlan20 -> Internet allow flow; flows=%+v", a.Flows)
	}
	if !hasService(db.Services, "tcp/5432") {
		t.Errorf("Vlan20 allow flow missing tcp/5432; got %v", db.Services)
	}

	// The explicit deny ip any any is drawn too.
	if findFlow(a.Flows, "Ethernet1", internetNode, "deny") == nil {
		t.Errorf("missing Ethernet1 -> Internet deny flow")
	}
}

func hasService(svcs []string, want string) bool {
	for _, s := range svcs {
		if s == want {
			return true
		}
	}
	return false
}
