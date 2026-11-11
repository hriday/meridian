package fwconfig

import "testing"

// A realistic `mgmt_cli show access-rulebase --format json --details-level full`
// export: a nested access-section, rules whose source/destination/service are
// resolved objects ({name,type}), Accept/Drop actions, and an objects-dictionary
// carrying the host/network/service details.
const cpSample = `{
  "rulebase": [
    {
      "type": "access-section",
      "name": "Inbound",
      "rulebase": [
        {
          "type": "access-rule",
          "name": "Web to DMZ",
          "rule-number": 1,
          "source": [ { "name": "Any", "type": "CpmiAnyObject" } ],
          "destination": [ { "name": "web-srv", "type": "host" } ],
          "service": [ { "name": "https", "type": "service-tcp" } ],
          "action": { "name": "Accept", "type": "RulebaseAction" }
        }
      ]
    },
    {
      "type": "access-rule",
      "name": "DMZ to Internal",
      "rule-number": 2,
      "source": [ { "name": "dmz-net", "type": "network" } ],
      "destination": [ { "name": "internal-net", "type": "network" } ],
      "service": [ { "name": "ssh", "type": "service-tcp" } ],
      "action": { "name": "Drop" }
    },
    {
      "type": "access-rule",
      "name": "Cleanup",
      "rule-number": 3,
      "source": [ { "name": "Any", "type": "CpmiAnyObject" } ],
      "destination": [ { "name": "Any", "type": "CpmiAnyObject" } ],
      "service": [ { "name": "Any", "type": "CpmiAnyObject" } ],
      "action": { "name": "Drop" }
    }
  ],
  "objects-dictionary": [
    { "uid": "a1", "name": "web-srv", "type": "host", "ipv4-address": "203.0.113.10" },
    { "uid": "a2", "name": "dmz-net", "type": "network", "subnet4": "10.1.10.0", "mask-length4": 24 },
    { "uid": "a3", "name": "internal-net", "type": "network", "subnet4": "10.1.0.0", "mask-length4": 16 },
    { "uid": "s1", "name": "https", "type": "service-tcp", "port": "443" },
    { "uid": "s2", "name": "ssh", "type": "service-tcp", "port": "22" }
  ]
}`

func TestAnalyzeCheckpointEmpty(t *testing.T) {
	if _, err := analyzeCheckpoint("   "); err == nil {
		t.Fatal("expected an error for empty config")
	}
	if _, err := analyzeCheckpoint("not json"); err == nil {
		t.Fatal("expected an error for non-JSON input")
	}
}

func TestAnalyzeCheckpointCounts(t *testing.T) {
	as, err := analyzeCheckpoint(cpSample)
	if err != nil {
		t.Fatalf("analyzeCheckpoint: %v", err)
	}
	if len(as) != 1 {
		t.Fatalf("want 1 analysis, got %d", len(as))
	}
	a := as[0]
	if a.Counts.Policies != 3 { // the section's rule is flattened in
		t.Errorf("policies: want 3, got %d", a.Counts.Policies)
	}
	if a.Counts.Addresses != 3 {
		t.Errorf("addresses: want 3, got %d", a.Counts.Addresses)
	}
	if a.Counts.Services != 2 {
		t.Errorf("services: want 2, got %d", a.Counts.Services)
	}
	if a.Counts.SubnetsIPs != 3 {
		t.Errorf("subnetsIps: want 3, got %d", a.Counts.SubnetsIPs)
	}
}

func TestAnalyzeCheckpointFlows(t *testing.T) {
	as, err := analyzeCheckpoint(cpSample)
	if err != nil {
		t.Fatalf("analyzeCheckpoint: %v", err)
	}
	a := as[0]

	// "Web to DMZ": source Any → Internet, to web-srv, accepted.
	in := findFlow(a.Flows, internetNode, "web-srv", "allow")
	if in == nil {
		t.Fatalf("missing Internet -> web-srv allow flow; flows=%+v", a.Flows)
	}
	if !hasService(in.Services, "https") {
		t.Errorf("Internet -> web-srv flow missing https; got %v", in.Services)
	}

	// "DMZ to Internal": object → object, dropped.
	d := findFlow(a.Flows, "dmz-net", "internal-net", "deny")
	if d == nil {
		t.Fatalf("missing dmz-net -> internal-net deny flow; flows=%+v", a.Flows)
	}
	if !hasService(d.Services, "ssh") {
		t.Errorf("dmz-net -> internal-net flow missing ssh; got %v", d.Services)
	}

	// Cleanup: Any → Any, dropped.
	if findFlow(a.Flows, internetNode, internetNode, "deny") == nil {
		t.Errorf("missing Internet -> Internet deny (cleanup) flow")
	}
}

// An inline/ordered-layer rule is an access-rule that gates traffic into a
// nested layer; its own source/dest rule must still be emitted, and the inner
// rules descended into — not swallowed.
func TestAnalyzeCheckpointInlineLayer(t *testing.T) {
	cfg := `{
  "rulebase": [
    { "type": "access-rule", "name": "To-Inner-Layer", "rule-number": 5,
      "action": { "name": "Inner Layer", "type": "RulebaseAction" },
      "inline-layer": "uid-inner",
      "source": [ {"name":"Any","type":"CpmiAnyObject"} ],
      "destination": [ {"name":"dmz-net","type":"network"} ],
      "rulebase": [
        { "type": "access-rule", "name": "Inner-Allow", "rule-number": 1,
          "action": {"name":"Accept"},
          "source": [ {"name":"Any"} ],
          "destination": [ {"name":"web-srv","type":"host"} ],
          "service": [ {"name":"https"} ] }
      ]
    }
  ],
  "objects-dictionary": [
    { "uid":"a2","name":"dmz-net","type":"network","subnet4":"10.1.10.0","mask-length4":24 },
    { "uid":"a1","name":"web-srv","type":"host","ipv4-address":"203.0.113.10" },
    { "uid":"s1","name":"https","type":"service-tcp","port":"443" }
  ]
}`
	as, err := analyzeCheckpoint(cfg)
	if err != nil {
		t.Fatalf("analyzeCheckpoint: %v", err)
	}
	a := as[0]
	var hasParent, hasInner bool
	for _, p := range a.Policies {
		if p.Name == "To-Inner-Layer" {
			hasParent = true
		}
		if p.Name == "Inner-Allow" {
			hasInner = true
		}
	}
	if !hasParent {
		t.Error("inline-layer parent rule 'To-Inner-Layer' was dropped")
	}
	if !hasInner {
		t.Error("inner-layer rule 'Inner-Allow' missing")
	}
	// The gating rule (Any → dmz-net) is not a drop, so it draws as allow.
	if findFlow(a.Flows, internetNode, "dmz-net", "allow") == nil {
		t.Errorf("missing Internet -> dmz-net flow from the inline-layer parent; flows=%+v", a.Flows)
	}
}

// A non-full export gives action/source/destination as bare UID strings; the
// action UID must be resolved through the dictionary, or an Accept reads as deny.
func TestAnalyzeCheckpointActionUidResolved(t *testing.T) {
	cfg := `{
  "rulebase": [
    { "type": "access-rule", "name": "Permit", "rule-number": 1,
      "source": [ "uid-src" ], "destination": [ "uid-dst" ],
      "service": [ {"name":"https"} ], "action": "uid-accept" }
  ],
  "objects-dictionary": [
    { "uid":"uid-src","name":"lan","type":"network","subnet4":"10.0.0.0","mask-length4":8 },
    { "uid":"uid-dst","name":"web-srv","type":"host","ipv4-address":"203.0.113.10" },
    { "uid":"uid-accept","name":"Accept","type":"RulebaseAction" },
    { "uid":"s1","name":"https","type":"service-tcp","port":"443" }
  ]
}`
	as, err := analyzeCheckpoint(cfg)
	if err != nil {
		t.Fatalf("analyzeCheckpoint: %v", err)
	}
	a := as[0]
	if findFlow(a.Flows, "lan", "web-srv", "allow") == nil {
		t.Errorf("bare-UID Accept action not resolved to allow; flows=%+v", a.Flows)
	}
	if findFlow(a.Flows, "lan", "web-srv", "deny") != nil {
		t.Errorf("Accept rule mislabeled as deny")
	}
}

// Only objects the rulebase actually references count; management/global objects
// in the dictionary (install targets, All_Internet) must not inflate the lists.
func TestAnalyzeCheckpointCountsReferencedOnly(t *testing.T) {
	cfg := `{
  "rulebase": [
    { "type":"access-rule","name":"r1","rule-number":1,
      "source":[{"name":"lan","type":"network"}],
      "destination":[{"name":"web-srv","type":"host"}],
      "service":[{"name":"https","type":"service-tcp"}],
      "action":{"name":"Accept"} }
  ],
  "objects-dictionary": [
    {"uid":"a1","name":"lan","type":"network","subnet4":"10.0.0.0","mask-length4":8},
    {"uid":"a2","name":"web-srv","type":"host","ipv4-address":"203.0.113.10"},
    {"uid":"s1","name":"https","type":"service-tcp","port":"443"},
    {"uid":"g1","name":"Policy Targets","type":"simple-gateway","ipv4-address":"192.0.2.1"},
    {"uid":"g2","name":"All_Internet","type":"network","subnet4":"0.0.0.0","mask-length4":0}
  ]
}`
	as, _ := analyzeCheckpoint(cfg)
	a := as[0]
	if a.Counts.Addresses != 2 {
		t.Errorf("addresses: want 2 (lan, web-srv — not the install gateway or All_Internet), got %d (%+v)", a.Counts.Addresses, a.Addresses)
	}
}

// An export that is just the rules array (no wrapper object) is still valid JSON.
func TestAnalyzeCheckpointBareArray(t *testing.T) {
	cfg := `[ { "type":"access-rule","name":"r1","rule-number":1,
      "source":[{"name":"dmz-net","type":"network"}],
      "destination":[{"name":"Any"}],
      "service":[{"name":"https"}], "action":{"name":"Accept"} } ]`
	as, err := analyzeCheckpoint(cfg)
	if err != nil {
		t.Fatalf("bare rules array should parse: %v", err)
	}
	if as[0].Counts.Policies != 1 {
		t.Errorf("policies: want 1, got %d", as[0].Counts.Policies)
	}
}
