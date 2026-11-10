package fwconfig

// This file holds the analysis model types. The FortiGate interpreter
// (fortigate.go) fills VLANs/Zones/Addresses/Services/Policies; the flow
// deriver (flows.go) fills Flows and Counts.

// VLAN is a FortiGate VLAN interface (an interface carrying a vlanid).
type VLAN struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Interface string `json:"interface"`
	Subnet    string `json:"subnet"`
	Zone      string `json:"zone"`
}

// Zone groups interfaces.
type Zone struct {
	Name       string   `json:"name"`
	Interfaces []string `json:"interfaces"`
}

// Address is a firewall address object or group.
type Address struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`  // subnet | iprange | fqdn | geography | group
	Value string `json:"value"` // human-readable; never a secret
}

// Service is a firewall service or group.
type Service struct {
	Name  string `json:"name"`
	Proto string `json:"proto"` // TCP | UDP | ICMP | … | group
	Ports string `json:"ports"`
}

// Policy is a firewall policy, enough to derive flows and drill in.
type Policy struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	SrcIntf  []string `json:"srcIntf"`
	DstIntf  []string `json:"dstIntf"`
	SrcAddr  []string `json:"srcAddr"`
	DstAddr  []string `json:"dstAddr"`
	Services []string `json:"services"`
	Action   string   `json:"action"` // accept | deny
	Disabled bool     `json:"disabled"`
	Negated  bool     `json:"negated"`
}

// Flow is one VLAN-to-VLAN permitted/denied relationship, aggregated across
// policies by (Src, Dst, Action).
type Flow struct {
	Src      string   `json:"src"`
	Dst      string   `json:"dst"`
	Services []string `json:"services"`
	Action   string   `json:"action"` // allow | deny
}

// Counts are the headline numbers.
type Counts struct {
	VLANs      int `json:"vlans"`
	Policies   int `json:"policies"`
	Addresses  int `json:"addresses"`
	SubnetsIPs int `json:"subnetsIps"`
	Services   int `json:"services"`
	Zones      int `json:"zones"`
}

// Analysis is the result for one VDOM (or the whole config when single-VDOM).
type Analysis struct {
	VDOM      string    `json:"vdom"`
	VLANs     []VLAN    `json:"vlans"`
	Zones     []Zone    `json:"zones"`
	Addresses []Address `json:"addresses"`
	Services  []Service `json:"services"`
	Policies  []Policy  `json:"policies"`
	Flows     []Flow    `json:"flows"`
	Counts    Counts    `json:"counts"`
	Notes     []string  `json:"notes"`
}

// newAnalysis returns an Analysis with every slice initialised non-nil so JSON
// emits [] rather than null.
func newAnalysis(vdom string) Analysis {
	return Analysis{
		VDOM:      vdom,
		VLANs:     []VLAN{},
		Zones:     []Zone{},
		Addresses: []Address{},
		Services:  []Service{},
		Policies:  []Policy{},
		Flows:     []Flow{},
		Notes:     []string{},
	}
}
