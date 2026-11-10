// Package dnsaudit resolves a domain's core DNS and email records —
// nameservers, MX, SPF, DMARC, and DNSSEC — and grades each in plain language,
// with an overall posture summary. The classifiers (spfVerdict, dmarcVerdict)
// are pure; Audit does the network exchange via the dnsx seam. The CLI and
// Wails controller are thin adapters over Audit.
package dnsaudit

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/dnsx"
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

// queryFn is the DNS exchange, behind a package var so tests can inject a fake.
var queryFn = dnsx.Query

// Audit resolves domain's NS/MX/SPF/DMARC records and DNSSEC state via resolver
// and grades each. It errors if domain or resolver is empty. A DNS-level
// refusal (SERVFAIL/REFUSED) makes that check an error, never a false
// "missing" (dnsx returns no error for a response that arrives, so the rcode is
// inspected directly).
func Audit(ctx context.Context, domain, resolver string, timeout time.Duration) (Report, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return Report{}, errors.New("no domain to audit")
	}
	if strings.TrimSpace(resolver) == "" {
		return Report{}, errors.New("no resolver specified")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rep := Report{Domain: domain, Resolver: resolver}

	// Nameservers (also the query whose AD flag we read for DNSSEC).
	nsRes, nsErr := queryFn(ctx, domain, "NS", resolver)
	ns := Check{Name: "Nameservers"}
	switch {
	case nsErr != nil:
		ns.Status, ns.Note = StatusError, "Couldn't query NS: "+nsErr.Error()
	case refused(nsRes.Rcode):
		ns.Status, ns.Note = StatusError, "The resolver returned "+nsRes.Rcode+" for NS — the check couldn't be performed."
	default:
		ns.Records = values(nsRes, "NS")
		if len(ns.Records) == 0 {
			ns.Status, ns.Note = StatusMissing, "No NS records — the domain has no nameservers (or doesn't exist)."
		} else {
			ns.Status, ns.Note = StatusOK, "Served by "+strconv.Itoa(len(ns.Records))+" nameserver(s)."
		}
	}
	rep.Checks = append(rep.Checks, ns)

	// Mail (MX).
	mxRes, mxErr := queryFn(ctx, domain, "MX", resolver)
	mx := Check{Name: "Mail (MX)"}
	switch {
	case mxErr != nil:
		mx.Status, mx.Note = StatusError, "Couldn't query MX: "+mxErr.Error()
	case refused(mxRes.Rcode):
		mx.Status, mx.Note = StatusError, "The resolver returned "+mxRes.Rcode+" for MX — the check couldn't be performed."
	default:
		mx.Records = values(mxRes, "MX")
		switch {
		case len(mx.Records) == 0:
			mx.Status, mx.Note = StatusWarn, "No MX records — this domain can't receive mail."
		case isNullMX(mx.Records):
			mx.Status, mx.Note = StatusWarn, "Null MX (0 .) — the domain explicitly accepts no mail, by design."
		default:
			mx.Status, mx.Note = StatusOK, "Accepts mail via "+strconv.Itoa(len(mx.Records))+" host(s)."
		}
	}
	rep.Checks = append(rep.Checks, mx)

	// SPF (apex TXT).
	txtRes, txtErr := queryFn(ctx, domain, "TXT", resolver)
	spf := Check{Name: "SPF"}
	if txtErr != nil {
		spf.Status, spf.Note = StatusError, "Couldn't query TXT: "+txtErr.Error()
	} else if refused(txtRes.Rcode) {
		spf.Status, spf.Note = StatusError, "The resolver returned "+txtRes.Rcode+" for TXT — the check couldn't be performed."
	} else {
		spf.Status, spf.Note, spf.Records = spfVerdict(values(txtRes, "TXT"))
	}
	rep.Checks = append(rep.Checks, spf)

	// DMARC (_dmarc TXT).
	dRes, dErr := queryFn(ctx, "_dmarc."+domain, "TXT", resolver)
	dmarc := Check{Name: "DMARC"}
	if dErr != nil {
		dmarc.Status, dmarc.Note = StatusError, "Couldn't query _dmarc TXT: "+dErr.Error()
	} else if refused(dRes.Rcode) {
		dmarc.Status, dmarc.Note = StatusError, "The resolver returned "+dRes.Rcode+" for _dmarc — the check couldn't be performed."
	} else {
		dmarc.Status, dmarc.Note, dmarc.Records = dmarcVerdict(values(dRes, "TXT"))
	}
	rep.Checks = append(rep.Checks, dmarc)

	// DNSSEC (AD flag from the NS response).
	sec := Check{Name: "DNSSEC"}
	switch {
	case nsErr != nil || refused(nsRes.Rcode):
		sec.Status, sec.Note = StatusError, "Couldn't determine DNSSEC — the apex query didn't complete."
	case nsRes.Authenticated:
		sec.Status, sec.Note = StatusOK, "Answers are DNSSEC-validated (your resolver set the AD flag)."
	default:
		sec.Status, sec.Note = StatusWarn, "Not DNSSEC-validated. This reflects your resolver: an unsigned zone and a non-validating resolver look the same, so this is a signal, not a proof."
	}
	rep.Checks = append(rep.Checks, sec)

	rep.Summary = summarize(&rep)
	return rep, nil
}

// refused reports whether a DNS rcode means the query was refused/failed rather
// than answered (NOERROR) or a clean "no such name" (NXDOMAIN).
func refused(rcode string) bool {
	return rcode != "" && rcode != "NOERROR" && rcode != "NXDOMAIN"
}

// values returns the answer values of the given record type, with surrounding
// quotes stripped (dnsx renders TXT values quoted).
func values(res dnsx.Result, rtype string) []string {
	var out []string
	for _, a := range res.Answers {
		if a.Type == rtype {
			out = append(out, strings.Trim(a.Value, `"`))
		}
	}
	return out
}

func isNullMX(records []string) bool {
	if len(records) != 1 {
		return false
	}
	f := strings.Fields(records[0])
	return len(f) == 2 && f[0] == "0" && (f[1] == "." || f[1] == "")
}

// summarize counts the grades and names the first thing to fix, in severity-
// ordered check order (NS, MX, SPF, DMARC, DNSSEC).
func summarize(r *Report) string {
	for _, c := range r.Checks {
		switch c.Status {
		case StatusOK:
			r.OKs++
		case StatusWarn:
			r.Warns++
		case StatusMissing:
			r.Missing++
		case StatusError:
			r.Errors++
		}
	}
	for _, c := range r.Checks {
		if c.Status != StatusOK {
			return c.Name + " needs attention — " + c.Note
		}
	}
	return "Posture looks healthy — nameservers, mail, SPF, DMARC and DNSSEC all check out."
}
