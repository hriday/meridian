// Package dnsaudit resolves a domain's core DNS and email records —
// nameservers, MX, SPF, DMARC, and DNSSEC — and grades each in plain language,
// with an overall posture summary. The classifiers (spfVerdict, dmarcVerdict)
// are pure; Audit does the network exchange via the dnsx seam. The CLI and
// Wails controller are thin adapters over Audit.
package dnsaudit

import (
	"strings"
)

// Status is a per-check grade.
type Status string

const (
	StatusOK      Status = "ok"      // healthy
	StatusWarn    Status = "warn"    // present but weak / permissive
	StatusMissing Status = "missing" // the record is absent
	StatusError   Status = "error"   // the lookup could not be performed
)

// Check is one audited aspect of the domain.
type Check struct {
	Name    string   `json:"name"`
	Status  Status   `json:"status"`
	Note    string   `json:"note"`
	Records []string `json:"records,omitempty"`
}

// Report is the full posture audit for one domain.
type Report struct {
	Domain   string  `json:"domain"`
	Resolver string  `json:"resolver"`
	Checks   []Check `json:"checks"`
	OKs      int     `json:"oks"`
	Warns    int     `json:"warns"`
	Missing  int     `json:"missing"`
	Errors   int     `json:"errors"`
	Summary  string  `json:"summary"`
}

// spfVerdict classifies a domain's apex TXT records (quotes already stripped).
// It returns the grade, a plain-language note, and the matched SPF record(s).
func spfVerdict(apexTXT []string) (Status, string, []string) {
	var spf []string
	for _, t := range apexTXT {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "v=spf1") {
			spf = append(spf, strings.TrimSpace(t))
		}
	}
	switch {
	case len(spf) == 0:
		return StatusMissing, "No SPF record — senders aren't constrained, so anyone can spoof mail from this domain.", nil
	case len(spf) > 1:
		return StatusWarn, "More than one v=spf1 record — SPF is invalid when duplicated (RFC 7208); keep exactly one.", spf
	}
	switch spfAllQualifier(spf[0]) {
	case "-":
		return StatusOK, "SPF ends in -all (hard fail) — unlisted senders are rejected.", spf
	case "~":
		return StatusOK, "SPF ends in ~all (soft fail) — unlisted senders are marked, not rejected; -all is stronger.", spf
	case "?":
		return StatusWarn, "SPF ends in ?all (neutral) — no enforcement; use -all or ~all.", spf
	case "+":
		return StatusWarn, "SPF ends in +all — it passes any sender, which defeats the point; use -all.", spf
	default:
		return StatusWarn, "SPF has no all mechanism — nothing is enforced for unlisted senders; add -all.", spf
	}
}

// spfAllQualifier returns the qualifier of the terminal all mechanism ("-",
// "~", "?", "+"), or "" if there is no all mechanism. A bare "all" defaults to
// "+".
func spfAllQualifier(record string) string {
	for _, tok := range strings.Fields(record) {
		low := strings.ToLower(tok)
		switch {
		case low == "all":
			return "+"
		case len(low) >= 4 && low[1:] == "all" && strings.ContainsRune("-~?+", rune(low[0])):
			return string(low[0])
		}
	}
	return ""
}

// dmarcVerdict classifies the _dmarc TXT records (quotes already stripped),
// returning the grade, a note, and the matched record(s).
func dmarcVerdict(txt []string) (Status, string, []string) {
	var dmarc []string
	for _, t := range txt {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "v=dmarc1") {
			dmarc = append(dmarc, strings.TrimSpace(t))
		}
	}
	if len(dmarc) == 0 {
		return StatusMissing, "No DMARC record — nothing tells receivers what to do with spoofed mail; add one at _dmarc.", nil
	}
	switch dmarcPolicy(dmarc[0]) {
	case "reject":
		return StatusOK, "DMARC p=reject — spoofed mail is rejected.", dmarc
	case "quarantine":
		return StatusOK, "DMARC p=quarantine — spoofed mail is quarantined; p=reject is stronger.", dmarc
	case "none":
		return StatusWarn, "DMARC p=none — monitoring only; it reports but enforces nothing. Move to quarantine or reject.", dmarc
	default:
		return StatusWarn, "DMARC record has no p= policy tag — it's malformed and won't be enforced.", dmarc
	}
}

// dmarcPolicy extracts the p= tag value (lowercased), or "" if absent.
func dmarcPolicy(record string) string {
	for _, part := range strings.Split(record, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "p=") {
			return strings.ToLower(strings.TrimSpace(part[2:]))
		}
	}
	return ""
}
