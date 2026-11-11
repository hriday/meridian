package fwconfig

import (
	"fmt"
	"net"
	"strings"
)

// analyzeArista reads an Arista EOS running-config and fills one Analysis. EOS
// is indentation-structured: `interface` and `ip access-list` headers own the
// indented lines beneath them. Named ACLs hold sequence-numbered permit/deny
// rules (CIDR / host / any) and are bound to an interface with
// `ip access-group <name> in|out`. Like ASA, an EOS ACL has no destination zone,
// so flows show the bound interface with "Internet (any)" standing in for the
// permitted destinations — a Note records this and each policy keeps the real
// source/destination addresses. Standard ACLs (source-only) are not interface
// traffic policies and are skipped.
func analyzeArista(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("arista: no config provided")
	}
	a := newAnalysis("")

	type ifaceAcc struct {
		name, subnet string
		bound        bool // has an `ip access-group` binding
	}
	var cur *ifaceAcc
	segByName := map[string]*VLAN{}
	var segOrder []string
	// Only L3/segment-bearing interfaces are segments: a real EOS config is
	// mostly L2 access ports and admin-down interfaces, which have neither an
	// `ip address` nor an ACL binding and must not pollute the zone list.
	flushIface := func() {
		if cur != nil && cur.name != "" && (cur.subnet != "" || cur.bound) {
			if _, ok := segByName[cur.name]; !ok {
				segOrder = append(segOrder, cur.name)
			}
			segByName[cur.name] = &VLAN{Name: cur.name, Interface: cur.name, Subnet: cur.subnet, Zone: cur.name}
		}
		cur = nil
	}

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
	bindings := map[string][]bind{} // acl name -> interface(s) + direction

	var curAcl string // name of the `ip access-list` block we're inside ("" if none)
	aclSkip := false  // true inside a `standard` ACL block

	for _, raw := range strings.Split(configText, "\n") {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || trimmed == "!" {
			flushIface()
			curAcl, aclSkip = "", false
			continue
		}
		indented := raw[0] == ' ' || raw[0] == '\t'
		toks := tokenize(trimmed)
		if len(toks) == 0 {
			continue
		}

		if !indented {
			flushIface()
			curAcl, aclSkip = "", false
			switch {
			case toks[0] == "interface" && len(toks) >= 2:
				cur = &ifaceAcc{name: toks[1]}
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
			}
			continue
		}

		switch {
		case cur != nil:
			switch toks[0] {
			case "ip":
				if len(toks) >= 3 && toks[1] == "address" {
					if strings.Contains(toks[2], "/") { // ignore `dhcp`, `unnumbered`
						cur.subnet = toks[2]
					}
				} else if len(toks) >= 4 && toks[1] == "access-group" {
					// ip access-group <name> in|out
					if d := toks[3]; d == "in" || d == "out" {
						bindings[toks[2]] = append(bindings[toks[2]], bind{intf: cur.name, dir: d})
						cur.bound = true
					}
				}
			}
		case curAcl != "" && !aclSkip:
			if p := parseAristaRule(toks, curAcl, &nextID); p != nil {
				addAce(curAcl, p)
			}
		}
	}
	flushIface()

	// Bind each ACL's rules to its interface(s). One ACL applied to several
	// interfaces is idiomatic EOS, so collect every `in` interface as a source
	// and every `out` interface as a destination — each yields its own flow.
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
	for _, acl := range aclOrder {
		for _, p := range aces[acl] {
			a.Policies = append(a.Policies, *p)
		}
	}
	if len(a.Policies) > 0 {
		a.Notes = append(a.Notes, "Arista EOS access lists are bound to an interface and have no destination zone; flows show the bound interface, with \"Internet (any)\" standing in for the destinations the rule permits — see each policy's destination addresses for specifics.")
	}

	derive(&a)
	return []Analysis{a}, nil
}

// parseAristaRule parses one EOS ACL rule into a Policy. A rule is an optional
// leading sequence number, then permit|deny, a protocol, a source spec, an
// optional source-port operator, a destination spec, and an optional
// destination-port operator. remark / counters / unparseable lines yield nil.
func parseAristaRule(toks []string, acl string, nextID *int) *Policy {
	i := 0
	if i < len(toks) && isSeqNum(toks[i]) { // leading sequence number
		i++
	}
	if i >= len(toks) {
		return nil
	}
	switch toks[i] {
	case "permit", "deny":
	default:
		return nil // remark, statistics, etc.
	}
	action := normalizeAction(toks[i])
	i++
	if i >= len(toks) {
		return nil
	}
	proto := strings.ToLower(toks[i])
	i++

	src := readAristaSpec(toks, &i)
	skipAristaSrcPort(toks, &i)
	dst := readAristaSpec(toks, &i)

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

// readAristaSpec reads one EOS address spec (any | host <ip> | <ip>/<len> |
// <ip> <wildcard>) advancing i past it.
func readAristaSpec(toks []string, i *int) string {
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
	default:
		t := toks[*i]
		if strings.Contains(t, "/") { // CIDR
			*i++
			return t
		}
		if net.ParseIP(t) != nil {
			if *i+1 < len(toks) && net.ParseIP(toks[*i+1]) != nil { // ip + wildcard mask
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

// skipAristaSrcPort advances i past an optional source-port operator sitting
// between the source and destination specs. It never consumes an address token.
func skipAristaSrcPort(toks []string, i *int) {
	if *i >= len(toks) {
		return
	}
	switch toks[*i] {
	case "eq", "neq", "lt", "gt":
		if *i+1 < len(toks) {
			*i += 2
		}
	case "range":
		if *i+2 < len(toks) {
			*i += 3
		}
	}
}

// readAristaDstPort reads an optional destination-port operator after the dst
// spec, returning a readable port string and advancing i past it.
func readAristaDstPort(toks []string, i *int) string {
	if *i >= len(toks) {
		return ""
	}
	switch toks[*i] {
	case "eq":
		if *i+1 < len(toks) {
			v := toks[*i+1]
			*i += 2
			return v
		}
	case "neq":
		if *i+1 < len(toks) {
			v := "!" + toks[*i+1] // "every port except" — not the same as eq
			*i += 2
			return v
		}
	case "lt":
		if *i+1 < len(toks) {
			v := "<" + toks[*i+1]
			*i += 2
			return v
		}
	case "gt":
		if *i+1 < len(toks) {
			v := ">" + toks[*i+1]
			*i += 2
			return v
		}
	case "range":
		if *i+2 < len(toks) {
			v := toks[*i+1] + "-" + toks[*i+2]
			*i += 3
			return v
		}
	}
	return ""
}

// isSeqNum reports whether s is a non-empty all-digits ACL sequence number.
func isSeqNum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
