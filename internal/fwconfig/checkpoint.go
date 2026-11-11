package fwconfig

import (
	"encoding/json"
	"fmt"
	"strings"
)

// analyzeCheckpoint reads a Check Point access rulebase exported as JSON by
// `mgmt_cli show access-rulebase --format json --details-level full` and fills
// one Analysis. Unlike the CLI-config vendors, Check Point rules are
// object-to-object rather than interface-bound, so each rule's source and
// destination objects are the flow nodes — a "what talks to what" reading at the
// object/zone level — with "Any" standing in for the Internet node. Access
// sections nest a rulebase, which is flattened. The objects-dictionary supplies
// host/network/service details for the address and service lists; secrets are
// not part of a rulebase export and are never surfaced.
func analyzeCheckpoint(configText string) ([]Analysis, error) {
	s := strings.TrimSpace(configText)
	if s == "" {
		return nil, fmt.Errorf("checkpoint: no config provided")
	}
	var root cpRoot
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		return nil, fmt.Errorf("checkpoint: not valid JSON — export with `mgmt_cli show access-rulebase --format json --details-level full`: %w", err)
	}
	if len(root.Rulebase) == 0 {
		return nil, fmt.Errorf("checkpoint: no \"rulebase\" array found — is this a show-access-rulebase export?")
	}
	a := newAnalysis("")

	dict := map[string]cpObject{}
	for _, o := range root.Objects {
		if o.UID != "" {
			dict[o.UID] = o
		}
		if o.Name != "" {
			dict["name:"+o.Name] = o
		}
	}

	nextID := 0
	var walk func(items []json.RawMessage)
	walk = func(items []json.RawMessage) {
		for _, raw := range items {
			var e cpRuleEnvelope
			if err := json.Unmarshal(raw, &e); err != nil {
				continue
			}
			if e.Type == "access-section" || len(e.Rulebase) > 0 {
				walk(e.Rulebase) // flatten sections
				continue
			}
			if e.Type != "access-rule" {
				continue
			}
			src := cpRefNames(e.Source, dict)
			dst := cpRefNames(e.Destination, dict)
			action := "deny"
			if strings.EqualFold(cpActionName(e.Action), "accept") {
				action = "accept"
			}
			nextID++
			name := e.Name
			if name == "" {
				name = fmt.Sprintf("Rule %d", e.RuleNumber)
			}
			a.Policies = append(a.Policies, Policy{
				ID:       nextID,
				Name:     name,
				SrcIntf:  src, // object / zone names are the flow nodes
				DstIntf:  dst,
				SrcAddr:  src,
				DstAddr:  dst,
				Services: cpServiceNames(e.Service, dict),
				Action:   action,
				Negated:  e.SourceNegate || e.DestinationNegate,
			})
		}
	}
	walk(root.Rulebase)

	// Address, service and zone lists from the dictionary (skipping "Any").
	seenAddr, seenSvc := map[string]bool{}, map[string]bool{}
	for _, o := range root.Objects {
		if cpIsAny(o.Name) {
			continue
		}
		switch o.Type {
		case "host", "network", "address-range", "group", "group-with-exclusion",
			"simple-gateway", "CpmiGatewayCluster", "CpmiClusterMember", "gateway":
			if !seenAddr[o.Name] {
				seenAddr[o.Name] = true
				a.Addresses = append(a.Addresses, Address{Name: o.Name, Kind: cpAddrKind(o.Type), Value: cpAddrValue(o)})
			}
		case "service-tcp", "service-udp", "service-icmp", "service-icmp6", "service-other", "service-group":
			if !seenSvc[o.Name] {
				seenSvc[o.Name] = true
				a.Services = append(a.Services, Service{Name: o.Name, Proto: cpSvcProto(o.Type), Ports: o.Port})
			}
		case "security-zone":
			a.Zones = append(a.Zones, Zone{Name: o.Name, Interfaces: []string{o.Name}})
		}
	}

	derive(&a)
	return []Analysis{a}, nil
}

type cpRoot struct {
	Rulebase []json.RawMessage `json:"rulebase"`
	Objects  []cpObject        `json:"objects-dictionary"`
}

type cpObject struct {
	UID     string `json:"uid"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	IPv4    string `json:"ipv4-address"`
	Subnet4 string `json:"subnet4"`
	MaskLen int    `json:"mask-length4"`
	IPFirst string `json:"ipv4-address-first"`
	IPLast  string `json:"ipv4-address-last"`
	Port    string `json:"port"`
}

type cpRuleEnvelope struct {
	Type              string            `json:"type"`
	Name              string            `json:"name"`
	RuleNumber        int               `json:"rule-number"`
	Rulebase          []json.RawMessage `json:"rulebase"`
	Source            []json.RawMessage `json:"source"`
	Destination       []json.RawMessage `json:"destination"`
	Service           []json.RawMessage `json:"service"`
	Action            json.RawMessage   `json:"action"`
	SourceNegate      bool              `json:"source-negate"`
	DestinationNegate bool              `json:"destination-negate"`
}

// cpRefName resolves one source/destination element — either an inline object
// {"name":…} (details-level full) or a bare UID string (resolved via the
// dictionary) — to its object name.
func cpRefName(raw json.RawMessage, dict map[string]cpObject) string {
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Name != "" {
		return obj.Name
	}
	var uid string
	if err := json.Unmarshal(raw, &uid); err == nil {
		if o, ok := dict[uid]; ok && o.Name != "" {
			return o.Name
		}
		return uid
	}
	return ""
}

func cpRefNames(items []json.RawMessage, dict map[string]cpObject) []string {
	var out []string
	for _, raw := range items {
		if n := cpRefName(raw, dict); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// cpServiceNames is cpRefNames for services, dropping the "Any" service so it
// doesn't clutter a flow's service list.
func cpServiceNames(items []json.RawMessage, dict map[string]cpObject) []string {
	var out []string
	for _, raw := range items {
		n := cpRefName(raw, dict)
		if n == "" || cpIsAny(n) {
			continue
		}
		out = append(out, n)
	}
	return out
}

// cpActionName reads a rule's action, an object {"name":"Accept"} or a bare string.
func cpActionName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var obj struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Name != "" {
		return obj.Name
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

func cpIsAny(name string) bool { return strings.EqualFold(name, "Any") }

func cpAddrKind(t string) string {
	switch t {
	case "address-range":
		return "iprange"
	case "group", "group-with-exclusion":
		return "group"
	default:
		return "subnet"
	}
}

func cpAddrValue(o cpObject) string {
	switch o.Type {
	case "host", "simple-gateway", "CpmiGatewayCluster", "CpmiClusterMember", "gateway":
		if o.IPv4 != "" {
			return o.IPv4 + "/32"
		}
	case "network":
		if o.Subnet4 != "" {
			return fmt.Sprintf("%s/%d", o.Subnet4, o.MaskLen)
		}
	case "address-range":
		if o.IPFirst != "" {
			return o.IPFirst + "-" + o.IPLast
		}
	}
	return ""
}

func cpSvcProto(t string) string {
	switch t {
	case "service-tcp":
		return "TCP"
	case "service-udp":
		return "UDP"
	case "service-icmp", "service-icmp6":
		return "ICMP"
	case "service-group":
		return "group"
	default:
		return "other"
	}
}
