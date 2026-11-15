// Package lanrecon reconciles a live ARP sweep against the saved IP-space
// inventory: it turns two standalone tools (arpscan + ipspace) into one
// workflow — surfacing untracked hosts, MAC changes (possible rogues), and
// records that no longer answer, and importing/updating them on request.
//
// It's point-in-time: Reconcile is a pure diff over a sweep and the current
// records; Apply writes only when explicitly asked.
package lanrecon

import (
	"net"
	"sort"
	"strings"

	"github.com/hriday/meridian/internal/arpscan"
	"github.com/hriday/meridian/internal/ipspace"
)

// Category is how a live host relates to the inventory.
type Category string

const (
	New       Category = "new"        // answered the sweep, not in inventory
	InSync    Category = "in-sync"    // in inventory, MAC matches (or none recorded)
	MACChange Category = "mac-change" // in inventory, a different MAC answered — possible rogue/spoof
	Missing   Category = "missing"    // active host record in the swept range that didn't answer
)

// Finding is one reconciled host.
type Finding struct {
	IP       string   `json:"ip"`
	MAC      string   `json:"mac"`              // observed MAC (recorded MAC, for Missing)
	Vendor   string   `json:"vendor,omitempty"` // observed vendor
	Category Category `json:"category"`
	RecordID string   `json:"recordId,omitempty"`
	Name     string   `json:"name,omitempty"`     // inventory name, if any
	KnownMAC string   `json:"knownMac,omitempty"` // inventory's recorded MAC (MACChange only)
}

// Report is a whole reconciliation pass.
type Report struct {
	Subnet    string    `json:"subnet"`
	Findings  []Finding `json:"findings"`
	New       int       `json:"new"`
	InSync    int       `json:"inSync"`
	MACChange int       `json:"macChange"`
	Missing   int       `json:"missing"`
}

// Reconcile diffs a sweep against the current inventory records.
func Reconcile(res arpscan.Result, records []ipspace.Record) Report {
	type recInfo struct {
		rec ipspace.Record
		mac string
	}
	byIP := map[string]recInfo{}
	for _, r := range records {
		ip, ok := hostIP(r.Prefix)
		if !ok {
			continue
		}
		byIP[ip] = recInfo{rec: r, mac: macFromTags(r.Tags)}
	}

	var findings []Finding
	seen := map[string]bool{}
	for _, h := range res.Hosts {
		ip := normIP(h.IP)
		if ip == "" {
			continue
		}
		seen[ip] = true
		obs := normMAC(h.MAC)
		info, ok := byIP[ip]
		if !ok {
			findings = append(findings, Finding{IP: ip, MAC: obs, Vendor: h.Vendor, Category: New})
			continue
		}
		f := Finding{IP: ip, MAC: obs, Vendor: h.Vendor, RecordID: info.rec.ID, Name: info.rec.Name}
		if info.mac != "" && obs != "" && info.mac != obs {
			f.Category = MACChange
			f.KnownMAC = info.mac
		} else {
			f.Category = InSync
		}
		findings = append(findings, f)
	}

	// Active host records inside the swept range that didn't answer.
	_, ipnet, _ := net.ParseCIDR(res.Subnet)
	for ip, info := range byIP {
		if seen[ip] || info.rec.Status != ipspace.Active {
			continue
		}
		if ipnet != nil {
			if p := net.ParseIP(ip); p == nil || !ipnet.Contains(p) {
				continue
			}
		}
		findings = append(findings, Finding{IP: ip, MAC: info.mac, Category: Missing, RecordID: info.rec.ID, Name: info.rec.Name})
	}

	sort.Slice(findings, func(i, j int) bool {
		return ipLess(findings[i].IP, findings[j].IP)
	})
	return tally(res.Subnet, findings)
}

func tally(subnet string, findings []Finding) Report {
	r := Report{Subnet: subnet, Findings: findings}
	for _, f := range findings {
		switch f.Category {
		case New:
			r.New++
		case InSync:
			r.InSync++
		case MACChange:
			r.MACChange++
		case Missing:
			r.Missing++
		}
	}
	return r
}

// hostIP returns the canonical IP of a host-level prefix (/32, /128, or a bare
// address). Larger blocks return ok=false — we reconcile at the host level.
func hostIP(prefix string) (string, bool) {
	if ip, ipnet, err := net.ParseCIDR(prefix); err == nil {
		if ones, bits := ipnet.Mask.Size(); ones == bits {
			return normIP(ip.String()), true
		}
		return "", false
	}
	if p := net.ParseIP(prefix); p != nil {
		return normIP(prefix), true
	}
	return "", false
}

func normIP(s string) string {
	p := net.ParseIP(strings.TrimSpace(s))
	if p == nil {
		return ""
	}
	if v := p.To4(); v != nil {
		return v.String()
	}
	return p.String()
}

// macFromTags returns the normalized MAC stored in a record's "mac:" tag, if any.
func macFromTags(tags []string) string {
	for _, t := range tags {
		if v, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(t)), "mac:"); ok {
			return normMAC(v)
		}
	}
	return ""
}

func normMAC(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if hw, err := net.ParseMAC(s); err == nil {
		return hw.String() // canonical lowercase colon form
	}
	return strings.ToLower(s)
}

func ipLess(a, b string) bool {
	pa, pb := net.ParseIP(a), net.ParseIP(b)
	if pa == nil || pb == nil {
		return a < b
	}
	a4, b4 := pa.To16(), pb.To16()
	for i := range a4 {
		if a4[i] != b4[i] {
			return a4[i] < b4[i]
		}
	}
	return false
}
