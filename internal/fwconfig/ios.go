package fwconfig

import (
	"fmt"
	"net"
	"strings"
)

// analyzeIOS reads a Cisco IOS / IOS-XE / NX-OS running-config and fills one
// Analysis. These share an indentation-structured ACL model: `interface` and
// `ip access-list [extended] <name>` headers own the indented lines beneath
// them; NX-OS prefixes each ACE with a sequence number. Named ACLs hold
// permit/deny rules (any / host / CIDR / wildcard / object-group) and are bound
// to an interface with `ip access-group <name> in|out`. Flat numbered ACLs
// (`access-list 101 permit …`) are also read. Like ASA, an IOS ACL has no
// destination zone, so flows show the bound interface with "Internet (any)"
// standing in for the permitted destinations — a Note records this and each
// policy keeps the real addresses. Standard (source-only) ACLs are skipped, and
// secrets live in sections the parser never reads.
func analyzeIOS(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("ios: no config provided")
	}
	a := newAnalysis("")

	type ifaceAcc struct {
		name, subnet string
		bound        bool
	}
	var cur *ifaceAcc
	segByName := map[string]*VLAN{}
	var segOrder []string
	flushIface := func() {
		if cur != nil && cur.name != "" && (cur.subnet != "" || cur.bound) {
			if _, ok := segByName[cur.name]; !ok {
				segOrder = append(segOrder, cur.name)
			}
			segByName[cur.name] = &VLAN{Name: cur.name, Interface: cur.name, Subnet: cur.subnet, Zone: cur.name}
		}
		cur = nil
	}

	addrs := map[string]*Address{}
	var addrOrder []string
	curGroup := "" // name of the object-group network block we're inside

	aces := map[string][]*Policy{}
	var aclOrder []string
	nextID := 0
	addAce := func(acl string, p *Policy) {
		if _, ok := aces[acl]; !ok {
			aclOrder = append(aclOrder, acl)
		}
		aces[acl] = append(aces[acl], p)
	}

	type bind struct{ intf, dir string }
	bindings := map[string][]bind{}

	curAcl := ""
	aclSkip := false

	reset := func() { flushIface(); curAcl, aclSkip, curGroup = "", false, "" }

	for _, raw := range strings.Split(configText, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || trimmed == "!" {
			reset()
			continue
		}
		indented := raw[0] == ' ' || raw[0] == '\t'
		toks := tokenize(trimmed)
		if len(toks) == 0 {
			continue
		}

		if !indented {
			reset()
			switch {
			case toks[0] == "interface" && len(toks) >= 2:
				cur = &ifaceAcc{name: strings.Join(toks[1:], " ")}
			case toks[0] == "ip" && len(toks) >= 3 && toks[1] == "access-list":
				// ip access-list [standard|extended] NAME
				i := 2
				if toks[i] == "standard" {
					aclSkip = true
					i++
				} else if toks[i] == "extended" {
					i++
				}
				if i < len(toks) {
					curAcl = toks[i]
				}
			case toks[0] == "access-list" && len(toks) >= 2:
				// flat numbered ACL: access-list 101 permit tcp …
				acl := toks[1]
				if p := parseIOSRule(toks[2:], acl, &nextID); p != nil {
					addAce(acl, p)
				}
			case toks[0] == "object-group" && len(toks) >= 2:
				// object-group network NAME | object-group ip address NAME (NX-OS)
				name := toks[len(toks)-1]
				kind := toks[1]
				if kind == "network" || kind == "ip" {
					curGroup = name
					if _, ok := addrs[name]; !ok {
						addrs[name] = &Address{Name: name, Kind: "group"}
						addrOrder = append(addrOrder, name)
					}
				}
			}
			continue
		}

		switch {
		case cur != nil:
			switch toks[0] {
			case "ip":
				if len(toks) >= 4 && toks[1] == "address" && net.ParseIP(toks[2]) != nil {
					cur.subnet = ipMaskToCIDR(toks[2] + " " + toks[3])
				} else if len(toks) >= 4 && toks[1] == "access-group" {
					if d := toks[3]; d == "in" || d == "out" {
						bindings[toks[2]] = append(bindings[toks[2]], bind{intf: cur.name, dir: d})
						cur.bound = true
					}
				}
			}
		case curGroup != "":
			if m := iosGroupMember(toks); m != "" {
				g := addrs[curGroup]
				g.Value = appendCSV(g.Value, m)
			}
		case curAcl != "" && !aclSkip:
			if p := parseIOSRule(toks, curAcl, &nextID); p != nil {
				addAce(curAcl, p)
			}
		}
	}
	reset()

	for acl, bs := range bindings {
		var srcIntf, dstIntf []string
		for _, b := range bs {
			if b.dir == "in" {
				srcIntf = append(srcIntf, b.intf)
			} else {
				dstIntf = append(dstIntf, b.intf)
			}
		}
		for _, p := range aces[acl] {
			if len(srcIntf) > 0 {
				p.SrcIntf = srcIntf
			}
			if len(dstIntf) > 0 {
				p.DstIntf = dstIntf
			}
		}
	}

	for _, n := range segOrder {
		v := segByName[n]
		a.VLANs = append(a.VLANs, *v)
		a.Zones = append(a.Zones, Zone{Name: n, Interfaces: []string{n}})
	}
	for _, n := range addrOrder {
		a.Addresses = append(a.Addresses, *addrs[n])
	}
	for _, acl := range aclOrder {
		for _, p := range aces[acl] {
			a.Policies = append(a.Policies, *p)
		}
	}
	if len(a.Policies) > 0 {
		a.Notes = append(a.Notes, "Cisco IOS/NX-OS access lists are bound to an interface and have no destination zone; flows show the bound interface, with \"Internet (any)\" standing in for the destinations the rule permits — see each policy's destination addresses for specifics.")
	}

	derive(&a)
	return []Analysis{a}, nil
}

// parseIOSRule parses one IOS/NX-OS ACE into a Policy: an optional leading
// sequence number, then permit|deny, a protocol, a source spec, an optional
// source-port operator, a destination spec, and an optional destination-port
// operator. remark / counters / unparseable lines yield nil.
func parseIOSRule(toks []string, acl string, nextID *int) *Policy {
	i := 0
	if i < len(toks) && isSeqNum(toks[i]) {
		i++
	}
	if i >= len(toks) {
		return nil
	}
	switch toks[i] {
	case "permit", "deny":
	default:
		return nil
	}
	action := normalizeAction(toks[i])
	i++
	if i >= len(toks) {
		return nil
	}
	proto := strings.ToLower(toks[i])
	i++

	src := readIOSSpec(toks, &i)
	skipAristaSrcPort(toks, &i)
	dst := readIOSSpec(toks, &i)

	var services []string
	if proto != "" && proto != "ip" {
		if port := readAristaDstPort(toks, &i); port != "" {
			services = []string{proto + "/" + port}
		}
	}

	*nextID++
	return &Policy{
		ID:       *nextID,
		Name:     acl,
		SrcAddr:  []string{src},
		DstAddr:  []string{dst},
		Services: services,
		Action:   action,
	}
}

// readIOSSpec reads one IOS/NX-OS address spec (any | host <ip> | <ip>/<len> |
// <ip> <wildcard> | object-group <name> | addrgroup <name>) advancing i past it.
func readIOSSpec(toks []string, i *int) string {
	if *i >= len(toks) {
		return ""
	}
	switch toks[*i] {
	case "any":
		*i++
		return "any"
	case "host":
		if *i+1 < len(toks) {
			v := toks[*i+1] + "/32"
			*i += 2
			return v
		}
	case "object-group", "addrgroup", "addr-group":
		if *i+1 < len(toks) {
			v := toks[*i+1]
			*i += 2
			return v
		}
	default:
		t := toks[*i]
		if strings.Contains(t, "/") {
			*i++
			return t
		}
		if net.ParseIP(t) != nil {
			if *i+1 < len(toks) && net.ParseIP(toks[*i+1]) != nil { // ip + wildcard
				v := t + " " + toks[*i+1]
				*i += 2
				return v
			}
			*i++
			return t
		}
	}
	v := toks[*i]
	*i++
	return v
}

// iosGroupMember renders one object-group network member (host / subnet / CIDR /
// nested group) as a readable string, or "" for description/other lines.
func iosGroupMember(toks []string) string {
	switch toks[0] {
	case "host":
		if len(toks) >= 2 {
			return toks[1] + "/32"
		}
	case "group-object":
		if len(toks) >= 2 {
			return toks[1]
		}
	case "network-object":
		return asaNetworkObject(toks[1:])
	case "description", "!":
		return ""
	default:
		t := toks[0]
		if strings.Contains(t, "/") { // NX-OS: 10.0.0.0/24
			return t
		}
		if net.ParseIP(t) != nil {
			if len(toks) >= 2 && net.ParseIP(toks[1]) != nil { // IOS: 10.0.0.0 255.255.255.0
				return ipMaskToCIDR(t + " " + toks[1])
			}
			return t + "/32"
		}
	}
	return ""
}
