package fwconfig

import "testing"

// A realistic pfSense/OPNsense config.xml excerpt: interface definitions with
// friendly descriptions, and <filter> rules (pass/block, interface-bound, with
// <any>/<network>/<address> endpoints, a port, and a disabled rule).
const pfsenseSample = `<?xml version="1.0"?>
<pfsense>
  <version>21.7</version>
  <interfaces>
    <wan>
      <if>em0</if>
      <descr>WAN</descr>
      <ipaddr>dhcp</ipaddr>
    </wan>
    <lan>
      <if>em1</if>
      <descr>LAN</descr>
      <ipaddr>10.1.0.1</ipaddr>
      <subnet>24</subnet>
    </lan>
    <opt1>
      <if>em2</if>
      <descr>DMZ</descr>
      <ipaddr>10.1.10.1</ipaddr>
      <subnet>24</subnet>
    </opt1>
  </interfaces>
  <filter>
    <rule>
      <type>pass</type>
      <interface>wan</interface>
      <protocol>tcp</protocol>
      <source><any></any></source>
      <destination><address>10.1.10.10</address><port>443</port></destination>
      <descr>Allow HTTPS to web</descr>
    </rule>
    <rule>
      <type>block</type>
      <interface>lan</interface>
      <protocol>tcp</protocol>
      <source><network>lan</network></source>
      <destination><any></any><port>3389</port></destination>
      <descr>Block RDP out</descr>
    </rule>
    <rule>
      <disabled></disabled>
      <type>pass</type>
      <interface>opt1</interface>
      <source><any></any></source>
      <destination><any></any></destination>
      <descr>disabled passthrough</descr>
    </rule>
  </filter>
</pfsense>`

func TestAnalyzePfsenseEmpty(t *testing.T) {
	if _, err := analyzePfsense("   "); err == nil {
		t.Fatal("expected an error for empty config")
	}
	if _, err := analyzePfsense("set firewall name X rule 1 action accept"); err == nil {
		t.Fatal("expected an error for non-XML input")
	}
}

func TestAnalyzePfsenseCounts(t *testing.T) {
	as, err := analyzePfsense(pfsenseSample)
	if err != nil {
		t.Fatalf("analyzePfsense: %v", err)
	}
	if len(as) != 1 {
		t.Fatalf("want 1 analysis, got %d", len(as))
	}
	a := as[0]
	if a.Counts.Policies != 3 { // incl. the disabled rule
		t.Errorf("policies: want 3, got %d", a.Counts.Policies)
	}
	if a.Counts.Zones != 3 { // WAN, LAN, DMZ
		t.Errorf("zones: want 3, got %d", a.Counts.Zones)
	}
}

func TestAnalyzePfsenseFlows(t *testing.T) {
	as, err := analyzePfsense(pfsenseSample)
	if err != nil {
		t.Fatalf("analyzePfsense: %v", err)
	}
	a := as[0]

	// WAN pass rule → WAN interface, HTTPS. Friendly name from <descr>.
	allow := findFlow(a.Flows, "WAN", internetNode, "allow")
	if allow == nil {
		t.Fatalf("missing WAN -> Internet allow flow; flows=%+v", a.Flows)
	}
	if !hasService(allow.Services, "tcp/443") {
		t.Errorf("WAN allow flow missing tcp/443; got %v", allow.Services)
	}

	// LAN block rule.
	if findFlow(a.Flows, "LAN", internetNode, "deny") == nil {
		t.Errorf("missing LAN -> Internet deny flow; flows=%+v", a.Flows)
	}

	// The DMZ rule is disabled, so it must not produce a flow.
	if findFlow(a.Flows, "DMZ", internetNode, "allow") != nil {
		t.Errorf("disabled DMZ rule should not produce a flow")
	}
}

// A <type>match</type> rule is a non-terminating classifier (shaper/queue), not
// an allow or deny; it must not be drawn as a block flow.
func TestAnalyzePfsenseMatchRuleNotDeny(t *testing.T) {
	cfg := `<?xml version="1.0"?><pfsense>
  <interfaces><lan><if>em1</if><descr>LAN</descr></lan></interfaces>
  <filter>
    <rule><type>match</type><interface>lan</interface>
      <source><any></any></source><destination><any></any></destination>
      <descr>shaper classifier</descr></rule>
  </filter>
</pfsense>`
	as, err := analyzePfsense(cfg)
	if err != nil {
		t.Fatalf("analyzePfsense: %v", err)
	}
	if findFlow(as[0].Flows, "LAN", internetNode, "deny") != nil {
		t.Errorf("a <type>match</type> classifier must not draw a deny flow")
	}
}

// A <not/> negation inverts the match; the model turns a negated policy into a
// Note rather than a misleading literal flow.
func TestAnalyzePfsenseNegationNoted(t *testing.T) {
	cfg := `<?xml version="1.0"?><pfsense>
  <interfaces><lan><if>em1</if><descr>LAN</descr></lan></interfaces>
  <filter>
    <rule><type>pass</type><interface>lan</interface>
      <source><any></any></source>
      <destination><not></not><network>10.0.0.0/8</network></destination>
      <descr>allow except internal</descr></rule>
  </filter>
</pfsense>`
	as, err := analyzePfsense(cfg)
	if err != nil {
		t.Fatalf("analyzePfsense: %v", err)
	}
	a := as[0]
	if findFlow(a.Flows, "LAN", internetNode, "allow") != nil {
		t.Errorf("a negated (<not>) rule should become a Note, not a literal allow flow")
	}
	if len(a.Policies) != 1 || !a.Policies[0].Negated {
		t.Errorf("policy should be marked Negated; got %+v", a.Policies)
	}
}
