// Package dnsaudit resolves a domain's core DNS and email records —
// nameservers, MX, SPF, DMARC, CAA, and DNSSEC — and grades each in plain language,
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

// caaVerdict classifies a domain's CAA records (which CAs may issue certificates
// for it). Absence is a soft warning, not "missing": CAA is optional, but
// without it any CA in the world may issue for the domain. A record naming
// issuers is healthy. Records arrive in dnsx string form, e.g. `0 issue
// "letsencrypt.org"`.
func caaVerdict(records []string) (Status, string, []string) {
	if len(records) == 0 {
		return StatusWarn, "No CAA record — any certificate authority may issue a certificate for this domain. Add CAA to limit issuance to the CAs you actually use.", nil
	}
	var issuers []string
	seen := map[string]bool{}
	forbidAll, hasIodef := false, false
	for _, r := range records {
		tag, val := parseCAA(r)
		switch tag {
		case "issue", "issuewild":
			ca := strings.TrimSpace(strings.SplitN(val, ";", 2)[0]) // drop CA params
			switch {
			case ca == "" || ca == ";":
				forbidAll = true
			case !seen[ca]:
				seen[ca] = true
				issuers = append(issuers, ca)
			}
		case "iodef":
			hasIodef = true
		}
	}
	var note string
	switch {
	case len(issuers) > 0:
		note = "Issuance restricted to: " + strings.Join(issuers, ", ") + "."
	case forbidAll:
		note = "CAA forbids all certificate issuance (issue \";\")."
	default:
		note = "CAA present but names no issuer."
	}
	if hasIodef {
		note += " Policy violations are reported via iodef."
	}
	return StatusOK, note, records
}

// parseCAA splits a CAA record string ("<flag> <tag> \"<value>\"") into its
// lowercased tag and unquoted value.
func parseCAA(record string) (tag, value string) {
	f := strings.Fields(record)
	if len(f) < 3 {
		return "", ""
	}
	return strings.ToLower(f[1]), strings.Trim(strings.Join(f[2:], " "), `"`)
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

	// CAA (which CAs may issue certificates for the domain).
	caaRes, caaErr := queryFn(ctx, domain, "CAA", resolver)
	caa := Check{Name: "CAA"}
	if caaErr != nil {
		caa.Status, caa.Note = StatusError, "Couldn't query CAA: "+caaErr.Error()
	} else if refused(caaRes.Rcode) {
		caa.Status, caa.Note = StatusError, "The resolver returned "+caaRes.Rcode+" for CAA — the check couldn't be performed."
	} else {
		caa.Status, caa.Note, caa.Records = caaVerdict(values(caaRes, "CAA"))
	}
	rep.Checks = append(rep.Checks, caa)

	// DNSSEC — validate the chain ourselves (DNSKEY + DS + RRSIG), rather than
	// trusting the resolver's AD flag, which can't tell an unsigned zone from a
	// non-validating resolver.
	sec := Check{Name: "DNSSEC"}
	if secRes, secErr := dnsx.ValidateDNSSEC(ctx, domain, resolver); secErr != nil {
		sec.Status, sec.Note = StatusError, "Couldn't validate DNSSEC: "+secErr.Error()
	} else {
		sec.Status, sec.Note = gradeDNSSEC(secRes)
	}
	rep.Checks = append(rep.Checks, sec)

	rep.Summary = summarize(&rep)
	return rep, nil
}

// gradeDNSSEC turns a validation result into a grade and a plain-language note,
// distinguishing "not signed" (common, mild) from "signed but unanchored" and
// "signed but broken" (which breaks resolution for validators), and never
// claiming more than was actually verified.
func gradeDNSSEC(r dnsx.DNSSECResult) (Status, string) {
	alg := ""
	if r.Algorithm != "" {
		alg = " (" + r.Algorithm + ")"
	}
	switch {
	case !r.Signed:
		return StatusWarn, "Zone is not DNSSEC-signed — responses can't be cryptographically verified. Signing it lets resolvers detect tampered answers."
	case !r.HasDS:
		return StatusWarn, "Zone is signed but the parent publishes no DS record — the chain of trust isn't anchored, so validators treat it as insecure. Add a DS record at the registrar."
	case !r.SecureDelegation:
		return StatusWarn, "DNSSEC appears broken — the parent DS matches no current DNSKEY, so validating resolvers may return SERVFAIL. Re-check the DS after any key rollover."
	case r.SignaturesValid:
		return StatusOK, "DNSSEC validated" + alg + " — signed zone, the parent DS matches a DNSKEY, and the key signatures verify."
	default:
		note := "Delegation is secure (the DS matches a DNSKEY), but the DNSKEY signature couldn't be verified here"
		if r.Detail != "" {
			note += " — " + r.Detail
		}
		return StatusWarn, note + "."
	}
}

// refused reports whether a DNS rcode means the query was refused/failed rather
// than answered (NOERROR) or a clean "no such name" (NXDOMAIN).
func refused(rcode string) bool {
	return rcode != "" && rcode != "NOERROR" && rcode != "NXDOMAIN"
}

// values returns the answer values of the given record type. TXT values are
// de-quoted: dnsx renders a multi-string TXT record as `"chunk1" "chunk2"`, but
// the real value is the chunks concatenated with no separator (a >255-byte SPF
// record is split this way), so the inter-chunk `" "` is removed before the
// outer quotes — otherwise an embedded quote could glue onto the terminal SPF
// mechanism and flip its grade.
func values(res dnsx.Result, rtype string) []string {
	var out []string
	for _, a := range res.Answers {
		if a.Type != rtype {
			continue
		}
		v := a.Value
		if rtype == "TXT" {
			v = strings.Trim(strings.ReplaceAll(v, `" "`, ""), `"`)
		}
		out = append(out, v)
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

// summarize counts the grades and names the thing to fix first — the most
// severe non-ok check (a missing record or a failed lookup outranks a mere
// warning, so a deliberate null MX never headlines over an absent DMARC), with
// ties broken by check order (NS, MX, SPF, DMARC, DNSSEC).
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
	best, bestRank := -1, 0
	for i, c := range r.Checks {
		if rk := severityRank(c.Status); rk > bestRank {
			best, bestRank = i, rk
		}
	}
	if best >= 0 {
		return r.Checks[best].Name + " needs attention — " + r.Checks[best].Note
	}
	return "Posture looks healthy — nameservers, mail, SPF, DMARC, CAA and DNSSEC all check out."
}

// severityRank ranks grades for the "fix first" headline: a missing record or a
// failed lookup (both "address this") outrank a soft warning; ok is 0.
func severityRank(s Status) int {
	switch s {
	case StatusMissing, StatusError:
		return 2
	case StatusWarn:
		return 1
	default:
		return 0
	}
}
