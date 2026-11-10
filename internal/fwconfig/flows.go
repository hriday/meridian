package fwconfig

import (
	"fmt"
	"sort"
	"strings"
)

const internetNode = "Internet (any)"

// derive fills a.Flows (from a.Policies) and a.Counts. Flows are a VLAN-level
// reading of the ruleset: each policy's source/dest interface maps to a node
// (VLAN/interface name, or "Internet (any)" for `any`/WAN), aggregated by
// (Src, Dst, Action) with services unioned. Disabled policies are skipped;
// negated / internet-service policies become a Note instead of a flow.
func derive(a *Analysis) {
	zoneMembers := map[string][]string{}
	for _, z := range a.Zones {
		zoneMembers[z.Name] = z.Interfaces
	}

	resolveOne := func(intf string) string {
		if intf == "" || strings.EqualFold(intf, "any") || a.wan[intf] {
			return internetNode
		}
		return intf
	}
	resolve := func(intf string) []string {
		if members, ok := zoneMembers[intf]; ok {
			var out []string
			for _, m := range members {
				out = append(out, resolveOne(m))
			}
			return out
		}
		return []string{resolveOne(intf)}
	}

	type key struct{ src, dst, action string }
	agg := map[key]*Flow{}
	var order []key
	svcSeen := map[key]map[string]bool{}

	addFlow := func(src, dst, action string, services []string) {
		k := key{src, dst, action}
		f, ok := agg[k]
		if !ok {
			f = &Flow{Src: src, Dst: dst, Action: action}
			agg[k] = f
			order = append(order, k)
			svcSeen[k] = map[string]bool{}
		}
		for _, s := range services {
			if s != "" && !svcSeen[k][s] {
				svcSeen[k][s] = true
				f.Services = append(f.Services, s)
			}
		}
	}

	for _, p := range a.Policies {
		if p.Disabled {
			continue
		}
		if p.Negated {
			a.Notes = append(a.Notes, fmt.Sprintf(
				"Policy %d (%s) uses address/service negation or internet-service — shown as a policy but not drawn as a flow.",
				p.ID, strings.TrimSpace(p.Name)))
			continue
		}
		action := "deny"
		if p.Action == "accept" {
			action = "allow"
		}
		var srcs, dsts []string
		for _, s := range p.SrcIntf {
			srcs = append(srcs, resolve(s)...)
		}
		for _, d := range p.DstIntf {
			dsts = append(dsts, resolve(d)...)
		}
		if len(srcs) == 0 {
			srcs = []string{internetNode}
		}
		if len(dsts) == 0 {
			dsts = []string{internetNode}
		}
		for _, s := range srcs {
			for _, d := range dsts {
				addFlow(s, d, action, p.Services)
			}
		}
	}

	for _, k := range order {
		a.Flows = append(a.Flows, *agg[k])
	}
	sort.SliceStable(a.Flows, func(i, j int) bool {
		if a.Flows[i].Src != a.Flows[j].Src {
			return a.Flows[i].Src < a.Flows[j].Src
		}
		if a.Flows[i].Dst != a.Flows[j].Dst {
			return a.Flows[i].Dst < a.Flows[j].Dst
		}
		return a.Flows[i].Action < a.Flows[j].Action
	})

	a.Counts = Counts{
		VLANs:      len(a.VLANs),
		Policies:   len(a.Policies),
		Addresses:  len(a.Addresses),
		Services:   len(a.Services),
		Zones:      len(a.Zones),
		SubnetsIPs: countSubnetsIPs(a),
	}
}

// countSubnetsIPs is the distinct set of subnet/host values across address
// objects (subnet + iprange) plus VLAN interface subnets, excluding the
// catch-all 0.0.0.0/0 and the "all" object.
func countSubnetsIPs(a *Analysis) int {
	seen := map[string]bool{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || v == "0.0.0.0/0" {
			return
		}
		seen[v] = true
	}
	for _, v := range a.VLANs {
		add(v.Subnet)
	}
	for _, ad := range a.Addresses {
		switch ad.Kind {
		case "subnet", "iprange":
			if strings.EqualFold(ad.Name, "all") {
				continue
			}
			add(ad.Value)
		}
	}
	return len(seen)
}
