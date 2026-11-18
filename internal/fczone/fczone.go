// Package fczone parses Fibre Channel zoning configuration from a switch and
// checks it for common SAN hygiene problems. It understands Cisco MDS (NX-OS /
// SAN-OS) output — `show zoneset [active]`, `show zone`, and running-config
// zoning — and Brocade Fabric OS `cfgshow` / `zoneshow`.
//
// The parser keys on keywords, not exact indentation (pasted configs vary), and
// records any line it doesn't recognize in Unparsed so coverage is honest rather
// than silently lossy.
package fczone

import (
	"regexp"
	"sort"
	"strings"
)

// Member is one zone or alias member.
type Member struct {
	Kind   string `json:"kind"`             // pwwn | fcid | fcalias | device-alias | interface | domain-port | wwn | name
	Value  string `json:"value"`            // the WWN / name / fcid / domain,port
	Active bool   `json:"active,omitempty"` // Cisco "*": the device is logged in
}

// Zone is a set of members allowed to talk to each other.
type Zone struct {
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

// Zoneset (Cisco) / cfg (Brocade) is a named collection of zones.
type Zoneset struct {
	Name   string   `json:"name"`
	Active bool     `json:"active"` // the enforced set (Cisco "active" / Brocade "effective")
	Zones  []string `json:"zones"`
}

// Alias (Cisco fcalias / Brocade alias) maps a friendly name to member(s).
type Alias struct {
	Name    string   `json:"name"`
	Members []Member `json:"members"`
}

// Finding is one hygiene observation.
type Finding struct {
	Severity string `json:"severity"` // warning | info
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Object   string `json:"object"`
}

// Report is the parsed, analysed configuration.
type Report struct {
	Vendor   string    `json:"vendor"` // "Cisco MDS" | "Brocade FOS" | "unknown"
	Zonesets []Zoneset `json:"zonesets"`
	Zones    []Zone    `json:"zones"`
	Aliases  []Alias   `json:"aliases"`
	Findings []Finding `json:"findings"`
	Counts   Counts    `json:"counts"`
	Unparsed []string  `json:"unparsed,omitempty"`
}

type Counts struct {
	Zonesets int `json:"zonesets"`
	Zones    int `json:"zones"`
	Aliases  int `json:"aliases"`
	Unparsed int `json:"unparsed"`
}

var (
	reCiscoZoneset  = regexp.MustCompile(`^zoneset\s+name\s+(\S+)`)
	reCiscoActivate = regexp.MustCompile(`^zoneset\s+activate\s+name\s+(\S+)`)
	reCiscoZone     = regexp.MustCompile(`^zone\s+name\s+(\S+)`)
	reCiscoFcalias  = regexp.MustCompile(`^fcalias\s+name\s+(\S+)`)
	reCiscoFcid     = regexp.MustCompile(`^(\*\s+)?fcid\s+(0x[0-9a-fA-F]+)\s*(?:\[(.+?)\])?`)
	reBrocadeLine   = regexp.MustCompile(`^(cfg|zone|alias):\s*(.*)$`)
	reWWN           = regexp.MustCompile(`^[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){7}$`)
	reDomainPort    = regexp.MustCompile(`^\d+,\d+$`)
)

// Parse auto-detects the vendor and parses + analyses the configuration.
func Parse(text string) Report {
	if looksBrocade(text) {
		return analyze(parseBrocade(text))
	}
	return analyze(parseCisco(text))
}

func looksBrocade(text string) bool {
	if strings.Contains(text, "Defined configuration:") || strings.Contains(text, "Effective configuration:") {
		return true
	}
	return reBrocadeLine.MatchString(firstMeaningfulLine(text))
}

func firstMeaningfulLine(text string) string {
	for _, l := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(l); t != "" {
			return t
		}
	}
	return ""
}

// --- Cisco MDS ------------------------------------------------------------

func parseCisco(text string) Report {
	r := Report{Vendor: "Cisco MDS"}
	activeMode := strings.Contains(text, "fcid 0x") // show zoneset active output

	var zs *Zoneset  // current zoneset
	var zone *Zone   // current zone
	var alias *Alias // current fcalias
	ctx := ""        // "zoneset" | "zone" | "fcalias"
	activateName := ""

	flushZone := func() {
		if zone != nil {
			r.Zones = append(r.Zones, *zone)
			zone = nil
		}
	}
	flushAlias := func() {
		if alias != nil {
			r.Aliases = append(r.Aliases, *alias)
			alias = nil
		}
	}

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		switch {
		case reCiscoActivate.MatchString(line):
			activateName = reCiscoActivate.FindStringSubmatch(line)[1]
		case reCiscoZoneset.MatchString(line):
			flushZone()
			flushAlias()
			name := reCiscoZoneset.FindStringSubmatch(line)[1]
			r.Zonesets = append(r.Zonesets, Zoneset{Name: name, Active: activeMode})
			zs = &r.Zonesets[len(r.Zonesets)-1]
			ctx = "zoneset"
		case reCiscoZone.MatchString(line):
			flushZone()
			flushAlias()
			name := reCiscoZone.FindStringSubmatch(line)[1]
			zone = &Zone{Name: name}
			if zs != nil {
				zs.Zones = append(zs.Zones, name)
			}
			ctx = "zone"
		case reCiscoFcalias.MatchString(line):
			flushZone()
			flushAlias()
			name := reCiscoFcalias.FindStringSubmatch(line)[1]
			alias = &Alias{Name: name}
			ctx = "fcalias"
		case reCiscoFcid.MatchString(line):
			m := reCiscoFcid.FindStringSubmatch(line)
			mem := Member{Kind: "fcid", Value: m[2], Active: strings.TrimSpace(m[1]) == "*"}
			if m[3] != "" { // bracket content like "pwwn 50:.." / "device-alias X"
				k, v := splitKindVal(m[3])
				mem.Kind, mem.Value = k, v
				mem.Active = strings.TrimSpace(m[1]) == "*"
			}
			addMember(zone, alias, ctx, mem)
		case strings.HasPrefix(line, "member "), strings.HasPrefix(line, "pwwn "),
			strings.HasPrefix(line, "fwwn "), strings.HasPrefix(line, "device-alias "),
			strings.HasPrefix(line, "fcalias "):
			body := strings.TrimPrefix(line, "member ")
			if ctx == "zoneset" {
				// a zoneset member is a zone name
				if zs != nil {
					zs.Zones = append(zs.Zones, strings.Fields(body)[0])
				}
				continue
			}
			k, v := splitKindVal(body)
			addMember(zone, alias, ctx, Member{Kind: k, Value: v})
		default:
			if !ciscoNoise(line) {
				r.Unparsed = append(r.Unparsed, line)
			}
		}
	}
	flushZone()
	flushAlias()
	if activateName != "" {
		for i := range r.Zonesets {
			if r.Zonesets[i].Name == activateName {
				r.Zonesets[i].Active = true
			}
		}
	}
	return r
}

func addMember(zone *Zone, alias *Alias, ctx string, m Member) {
	switch ctx {
	case "fcalias":
		if alias != nil {
			alias.Members = append(alias.Members, m)
		}
	default:
		if zone != nil {
			zone.Members = append(zone.Members, m)
		}
	}
}

// splitKindVal turns "pwwn 50:06:.." into ("pwwn","50:06:.."), handling the
// member keywords Cisco uses.
func splitKindVal(s string) (kind, val string) {
	f := strings.Fields(s)
	if len(f) == 0 {
		return "name", s
	}
	switch f[0] {
	case "pwwn", "fwwn":
		return "pwwn", get(f, 1)
	case "fcid":
		return "fcid", get(f, 1)
	case "fcalias":
		return "fcalias", get(f, 1)
	case "device-alias":
		return "device-alias", get(f, 1)
	case "interface":
		return "interface", strings.Join(f[1:], " ")
	case "domain-id":
		// "domain-id 1 portnumber 2"
		return "domain-port", strings.Join(f[1:], " ")
	default:
		if reWWN.MatchString(f[0]) {
			return "pwwn", f[0]
		}
		return "name", f[0]
	}
}

func get(f []string, i int) string {
	if i < len(f) {
		return f[i]
	}
	return ""
}

func ciscoNoise(line string) bool {
	// Prompts, separators and common non-zoning chatter.
	return strings.HasSuffix(line, "#") || strings.HasPrefix(line, "!") ||
		strings.HasPrefix(line, "Active zoneset") || strings.HasPrefix(line, "----")
}

// --- Brocade FOS ----------------------------------------------------------

func parseBrocade(text string) Report {
	r := Report{Vendor: "Brocade FOS"}
	section := "" // "defined" | "effective"
	var curKind, curName string
	var curMembers []string

	flush := func() {
		if curKind == "" {
			return
		}
		mem := parseBrocadeMembers(curMembers)
		switch curKind {
		case "cfg":
			zoneNames := make([]string, 0, len(mem))
			for _, m := range mem {
				zoneNames = append(zoneNames, m.Value)
			}
			upsertZoneset(&r, curName, zoneNames, section == "effective")
		case "zone":
			upsertZone(&r, curName, mem, section)
		case "alias":
			r.Aliases = append(r.Aliases, Alias{Name: curName, Members: mem})
		}
		curKind, curName, curMembers = "", "", nil
	}

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.HasPrefix(trimmed, "Defined configuration"):
			flush()
			section = "defined"
		case strings.HasPrefix(trimmed, "Effective configuration"):
			flush()
			section = "effective"
		case reBrocadeLine.MatchString(trimmed):
			flush()
			m := reBrocadeLine.FindStringSubmatch(trimmed)
			curKind = m[1]
			rest := strings.TrimSpace(m[2])
			fields := strings.SplitN(rest, " ", 2)
			curName = fields[0]
			if len(fields) > 1 {
				curMembers = append(curMembers, fields[1])
			}
		case isBrocadeNoise(trimmed):
			// ignore
		case curKind != "":
			// continuation of the current object's member list (wrapped line)
			curMembers = append(curMembers, trimmed)
		default:
			r.Unparsed = append(r.Unparsed, trimmed)
		}
	}
	flush()
	return r
}

func parseBrocadeMembers(chunks []string) []Member {
	var out []Member
	joined := strings.Join(chunks, " ")
	// Members are separated by ';' and/or whitespace.
	for _, part := range strings.FieldsFunc(joined, func(r rune) bool {
		return r == ';' || r == ' ' || r == '\t'
	}) {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		switch {
		case reWWN.MatchString(p):
			out = append(out, Member{Kind: "wwn", Value: strings.ToLower(p)})
		case reDomainPort.MatchString(p):
			out = append(out, Member{Kind: "domain-port", Value: p})
		default:
			out = append(out, Member{Kind: "name", Value: p})
		}
	}
	return out
}

func upsertZoneset(r *Report, name string, zones []string, active bool) {
	for i := range r.Zonesets {
		if r.Zonesets[i].Name == name {
			if len(zones) > 0 {
				r.Zonesets[i].Zones = zones
			}
			if active {
				r.Zonesets[i].Active = true
			}
			return
		}
	}
	r.Zonesets = append(r.Zonesets, Zoneset{Name: name, Zones: zones, Active: active})
}

func upsertZone(r *Report, name string, mem []Member, section string) {
	for i := range r.Zones {
		if r.Zones[i].Name == name {
			// Prefer the defined section's members (they keep alias names); only
			// fill from effective if we have nothing yet.
			if section == "defined" || len(r.Zones[i].Members) == 0 {
				r.Zones[i].Members = mem
			}
			return
		}
	}
	r.Zones = append(r.Zones, Zone{Name: name, Members: mem})
}

func isBrocadeNoise(line string) bool {
	low := strings.ToLower(line)
	return strings.Contains(low, "no configuration") || strings.HasSuffix(line, ">") ||
		strings.HasSuffix(line, "#") || strings.HasPrefix(line, "---")
}

// --- Analysis -------------------------------------------------------------

func analyze(r Report) Report {
	r.Counts = Counts{Zonesets: len(r.Zonesets), Zones: len(r.Zones), Aliases: len(r.Aliases), Unparsed: len(r.Unparsed)}

	// Which zones are referenced by at least one zoneset.
	inSet := map[string]bool{}
	activeZones := map[string]bool{}
	for _, zs := range r.Zonesets {
		for _, z := range zs.Zones {
			inSet[z] = true
			if zs.Active {
				activeZones[z] = true
			}
		}
	}
	// Which aliases are referenced by a zone.
	usedAlias := map[string]bool{}
	for _, z := range r.Zones {
		for _, m := range z.Members {
			if m.Kind == "fcalias" || m.Kind == "device-alias" || m.Kind == "name" {
				usedAlias[m.Value] = true
			}
		}
	}

	for _, z := range r.Zones {
		if strings.Contains(z.Name, "$default_zone$") {
			continue
		}
		switch n := len(z.Members); {
		case n == 0:
			r.Findings = append(r.Findings, Finding{"warning", "Empty zone", "The zone has no members — it does nothing and is likely stale.", z.Name})
		case n == 1:
			r.Findings = append(r.Findings, Finding{"warning", "Single-member zone", "A zone needs at least an initiator and a target; one member isolates the device.", z.Name})
		case n > 2:
			r.Findings = append(r.Findings, Finding{"warning", "Zone isn't single-initiator-single-target", "Has " + plural(n, "member") + ". Best practice is one initiator + one target per zone, so a misbehaving HBA can't disrupt others.", z.Name})
		}
		if !inSet[z.Name] {
			r.Findings = append(r.Findings, Finding{"warning", "Zone not in any zoneset", "Defined but not a member of any zoneset/cfg, so it is never enforced.", z.Name})
		} else if len(activeZones) > 0 && !activeZones[z.Name] {
			r.Findings = append(r.Findings, Finding{"info", "Zone not in the active set", "Defined and in a zoneset, but not the active/effective one.", z.Name})
		}
	}

	for _, a := range r.Aliases {
		if !usedAlias[a.Name] {
			r.Findings = append(r.Findings, Finding{"info", "Unused alias", "Defined but not referenced by any zone.", a.Name})
		}
	}

	activeCount := 0
	for _, zs := range r.Zonesets {
		if zs.Active {
			activeCount++
		}
	}
	if len(r.Zonesets) > 0 && activeCount == 0 {
		r.Findings = append(r.Findings, Finding{"info", "No active zoneset detected", "None of the zonesets is marked active/effective — paste `show zoneset active` or `cfgshow` to confirm what's enforced.", ""})
	}

	sort.SliceStable(r.Findings, func(i, j int) bool {
		rank := map[string]int{"warning": 0, "info": 1}
		if rank[r.Findings[i].Severity] != rank[r.Findings[j].Severity] {
			return rank[r.Findings[i].Severity] < rank[r.Findings[j].Severity]
		}
		return r.Findings[i].Object < r.Findings[j].Object
	})
	return r
}

func plural(n int, w string) string {
	if n == 1 {
		return "1 " + w
	}
	return itoa(n) + " " + w + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
