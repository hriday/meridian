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
