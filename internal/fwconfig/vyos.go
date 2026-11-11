package fwconfig

import (
	"fmt"
	"sort"
	"strings"
)

// analyzeVyos reads a VyOS firewall configuration in set-format (as produced by
// `show configuration commands`) and fills one Analysis. VyOS is zone-based:
// named rule-sets (`set firewall name <R> rule <N> …`) are wired to zone pairs
// with `set zone-policy zone <DEST> from <SRC> firewall name <R>`, so each rule
// becomes a SRC-zone → DEST-zone policy and zones resolve to their member
// interfaces for the flow, like SRX. A rule-set applied straight to an interface
// (`set interfaces ethernet <IF> firewall in|out name <R>`) maps to interface →
// Internet, like ASA. The 1.4 `firewall ipv4 name` spelling is accepted as an
// alias. Rule-set default-actions are not drawn as flows.
func analyzeVyos(configText string) ([]Analysis, error) {
	if strings.TrimSpace(configText) == "" {
		return nil, fmt.Errorf("vyos: no config provided")
	}

	rulesets := map[string]map[int]*vyRule{}
	var rsOrder []string
	getRule := func(rs string, n int) *vyRule {
		m, ok := rulesets[rs]
		if !ok {
			m = map[int]*vyRule{}
			rulesets[rs] = m
			rsOrder = append(rsOrder, rs)
		}
		r, ok := m[n]
		if !ok {
			r = &vyRule{}
			m[n] = r
		}
		return r
	}

	zoneIface := map[string][]string{}
	var zoneOrder []string
	ensureZone := func(z string) {
		if _, ok := zoneIface[z]; !ok {
			zoneIface[z] = nil
			zoneOrder = append(zoneOrder, z)
		}
	}
	addZoneIface := func(z, intf string) {
		ensureZone(z)
		zoneIface[z] = append(zoneIface[z], intf)
	}

	type vyBind struct{ ruleset, src, dst string }
	var binds []vyBind
	boundRS := map[string]bool{}

	for _, raw := range strings.Split(configText, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		toks := tokenize(line)
		if len(toks) < 2 || toks[0] != "set" {
			continue
		}
		toks = toks[1:]

		switch toks[0] {
		case "firewall":
			i := 1
			if i < len(toks) && (toks[i] == "ipv4" || toks[i] == "ipv6") {
				i++
			}
			if i+1 < len(toks) && toks[i] == "name" {
				vyParseRulesetLine(toks[i+1], toks[i+2:], getRule)
			}
		case "zone-policy":
			if len(toks) >= 3 && toks[1] == "zone" {
				z, rest := toks[2], toks[3:]
				switch {
				case len(rest) >= 2 && rest[0] == "interface":
					addZoneIface(z, rest[1])
				case len(rest) >= 5 && rest[0] == "from" && rest[2] == "firewall" && rest[3] == "name":
					ensureZone(z)
					ensureZone(rest[1])
					binds = append(binds, vyBind{ruleset: rest[4], src: rest[1], dst: z})
					boundRS[rest[4]] = true
				}
			}
		case "interfaces":
			// set interfaces <type> <IF> … firewall <in|out|local> name <R>
			fi := -1
			for i, t := range toks {
				if t == "firewall" {
					fi = i
					break
				}
			}
			if fi >= 2 && fi+3 < len(toks) && toks[fi+2] == "name" {
				intf, dir, rsn := toks[2], toks[fi+1], toks[fi+3]
				switch dir {
				case "in":
					binds = append(binds, vyBind{ruleset: rsn, src: intf})
					boundRS[rsn] = true
				case "out", "local":
					binds = append(binds, vyBind{ruleset: rsn, dst: intf})
					boundRS[rsn] = true
				}
			}
		}
	}

	a := newAnalysis("")
	for _, z := range zoneOrder {
		a.Zones = append(a.Zones, Zone{Name: z, Interfaces: zoneIface[z]})
	}

	nextID := 0
	emit := func(rs, src, dst string) {
		m := rulesets[rs]
		nums := make([]int, 0, len(m))
		for n := range m {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		for _, n := range nums {
			r := m[n]
			action := "deny"
			if r.action == "accept" {
				action = "accept"
			}
			p := Policy{
				ID:       nextInt(&nextID),
				Name:     fmt.Sprintf("%s/%d", rs, n),
				Action:   action,
				Disabled: r.disabled,
			}
			if src != "" {
				p.SrcIntf = []string{src}
			}
			if dst != "" {
				p.DstIntf = []string{dst}
			}
			if r.srcAddr != "" {
				p.SrcAddr = []string{r.srcAddr}
			}
			if r.dstAddr != "" {
				p.DstAddr = []string{r.dstAddr}
			}
			if s := vySvc(r.proto, r.dstPort); s != "" {
				p.Services = []string{s}
			}
			a.Policies = append(a.Policies, p)
		}
	}
	for _, b := range binds {
		emit(b.ruleset, b.src, b.dst)
	}
	for _, rs := range rsOrder { // rule-sets bound to nothing still get listed
		if !boundRS[rs] {
			emit(rs, "", "")
		}
	}

	derive(&a)
	return []Analysis{a}, nil
}

type vyRule struct {
	action, proto, srcAddr, dstAddr, dstPort string
	disabled                                 bool
}

// vyParseRulesetLine applies one `set firewall name <R> …` line (minus the
// `firewall [ipv4] name <R>` prefix) to the rule-set's rules.
func vyParseRulesetLine(rs string, rest []string, getRule func(string, int) *vyRule) {
	if len(rest) < 3 || rest[0] != "rule" {
		return // default-action, enable-default-log, … — not a numbered rule
	}
	r := getRule(rs, atoiSafe(rest[1]))
	f := rest[2:]
	switch f[0] {
	case "action":
		if len(f) >= 2 {
			r.action = f[1]
		}
	case "protocol":
		if len(f) >= 2 {
			r.proto = strings.ToLower(f[1])
		}
	case "disable":
		r.disabled = true
	case "source":
		if len(f) >= 3 && f[1] == "address" {
			r.srcAddr = f[2]
		}
	case "destination":
		if len(f) >= 3 && f[1] == "address" {
			r.dstAddr = f[2]
		} else if len(f) >= 3 && f[1] == "port" {
			r.dstPort = f[2]
		}
	}
}

func vySvc(proto, port string) string {
	switch {
	case port != "" && proto != "":
		return proto + "/" + port
	case port != "":
		return port
	case proto != "" && proto != "all" && proto != "ip":
		return proto
	default:
		return ""
	}
}

func nextInt(n *int) int { *n++; return *n }
