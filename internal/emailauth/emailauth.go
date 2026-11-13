// Package emailauth checks a domain's outbound-mail authentication — SPF, DKIM,
// and DMARC — from DNS alone, as a mail sender's-eye view: can this domain send
// authenticated mail, and how spoofable is it? It complements the broader DNS
// audit (which also grades SPF/DMARC) by adding DKIM selector discovery, which
// requires probing selector names DNS won't enumerate.
package emailauth

import (
	"context"
	"net/url"
	"strings"
)

// Status is a pass/warn/fail/none verdict for one mechanism.
type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// TXTLookup resolves the TXT records of a name; injected so tests stay offline.
type TXTLookup func(ctx context.Context, name string) ([]string, error)

// DefaultSelectors are common DKIM selectors to probe when the caller gives
// none — covers Google, Microsoft 365, and several ESPs.
var DefaultSelectors = []string{
	"google", "selector1", "selector2", "default", "k1", "k2", "k3",
	"mail", "dkim", "s1", "s2", "mandrill", "mxvault", "zoho", "pm", "ctct1",
}

type SPFResult struct {
	Status Status `json:"status"`
	Record string `json:"record"`
	Note   string `json:"note"`
}

type DMARCResult struct {
	Status Status `json:"status"`
	Record string `json:"record"`
	Policy string `json:"policy"` // none | quarantine | reject | ""
	Pct    string `json:"pct"`
	RUA    string `json:"rua"`
	Note   string `json:"note"`
}

type DKIMResult struct {
	Selector string `json:"selector"`
	Found    bool   `json:"found"`
	Record   string `json:"record"`
	Note     string `json:"note"`
}

type Report struct {
	Domain    string       `json:"domain"`
	SPF       SPFResult    `json:"spf"`
	DMARC     DMARCResult  `json:"dmarc"`
	DKIM      []DKIMResult `json:"dkim"`
	DKIMFound bool         `json:"dkimFound"`
	Summary   string       `json:"summary"`
}

// Check resolves and grades SPF, DMARC, and the given DKIM selectors (or the
// default set when none are supplied) for domain.
func Check(ctx context.Context, lookup TXTLookup, domain string, selectors []string) Report {
	d := normalizeDomain(domain)
	r := Report{Domain: d}
	r.SPF = checkSPF(ctx, lookup, d)
	r.DMARC = checkDMARC(ctx, lookup, d)

	if len(selectors) == 0 {
		selectors = DefaultSelectors
	}
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		dk := checkDKIM(ctx, lookup, d, sel)
		r.DKIM = append(r.DKIM, dk)
		if dk.Found {
			r.DKIMFound = true
		}
	}
	r.Summary = summarize(r)
	return r
}

func checkSPF(ctx context.Context, lookup TXTLookup, domain string) SPFResult {
	txt, _ := lookup(ctx, domain)
	var spf []string
	for _, t := range txt {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "v=spf1") {
			spf = append(spf, strings.TrimSpace(t))
		}
	}
	switch {
	case len(spf) == 0:
		return SPFResult{Status: StatusFail, Note: "No SPF record — anyone can send mail claiming to be this domain."}
	case len(spf) > 1:
		return SPFResult{Status: StatusFail, Record: strings.Join(spf, "  |  "),
			Note: "More than one SPF record — SPF is invalid when duplicated (RFC 7208); keep exactly one."}
	}
	rec := spf[0]
	switch spfAllQualifier(rec) {
	case "-":
		return SPFResult{Status: StatusPass, Record: rec, Note: "Ends in -all (hard fail) — unlisted senders are rejected."}
	case "~":
		return SPFResult{Status: StatusWarn, Record: rec, Note: "Ends in ~all (soft fail) — unlisted senders are flagged, not rejected; -all is stronger."}
	case "?":
		return SPFResult{Status: StatusWarn, Record: rec, Note: "Ends in ?all (neutral) — no enforcement; use -all or ~all."}
	case "+":
		return SPFResult{Status: StatusFail, Record: rec, Note: "Ends in +all — passes any sender, defeating SPF; use -all."}
	default:
		return SPFResult{Status: StatusWarn, Record: rec, Note: "No 'all' mechanism — unlisted senders aren't constrained; add -all."}
	}
}

// spfAllQualifier returns the qualifier of the terminal all mechanism.
func spfAllQualifier(rec string) string {
	fields := strings.Fields(strings.ToLower(rec))
	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		switch {
		case f == "all", f == "+all":
			return "+"
		case f == "-all":
			return "-"
		case f == "~all":
			return "~"
		case f == "?all":
			return "?"
		}
	}
	return ""
}

func checkDMARC(ctx context.Context, lookup TXTLookup, domain string) DMARCResult {
	txt, _ := lookup(ctx, "_dmarc."+domain)
	var rec string
	for _, t := range txt {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "v=dmarc1") {
			rec = strings.TrimSpace(t)
			break
		}
	}
	if rec == "" {
		return DMARCResult{Status: StatusFail, Note: "No DMARC record — receivers have no policy to act on failed SPF/DKIM, so spoofs can slip through."}
	}
	tags := parseTags(rec)
	policy := tags["p"]
	res := DMARCResult{Record: rec, Policy: policy, Pct: tags["pct"], RUA: tags["rua"]}
	switch policy {
	case "reject":
		res.Status, res.Note = StatusPass, "p=reject — failing mail is rejected (strongest)."
	case "quarantine":
		res.Status, res.Note = StatusPass, "p=quarantine — failing mail goes to spam; p=reject is stronger."
	case "none":
		res.Status, res.Note = StatusWarn, "p=none — monitor only; failing mail is still delivered. Move to quarantine/reject once reports look clean."
	default:
		res.Status, res.Note = StatusWarn, "DMARC present but no valid p= policy."
	}
	return res
}

func checkDKIM(ctx context.Context, lookup TXTLookup, domain, selector string) DKIMResult {
	name := selector + "._domainkey." + domain
	txt, _ := lookup(ctx, name)
	joined := strings.Join(txt, "")
	low := strings.ToLower(joined)
	if strings.Contains(low, "v=dkim1") || strings.Contains(low, "p=") {
		return DKIMResult{Selector: selector, Found: true, Record: joined, Note: "DKIM key published."}
	}
	return DKIMResult{Selector: selector, Found: false, Note: "No key at " + name}
}

func summarize(r Report) string {
	var parts []string
	parts = append(parts, "SPF "+string(r.SPF.Status))
	parts = append(parts, "DMARC "+string(r.DMARC.Status))
	if r.DKIMFound {
		parts = append(parts, "DKIM found")
	} else {
		parts = append(parts, "DKIM not found")
	}
	return strings.Join(parts, " · ")
}

// parseTags parses "k=v; k2=v2" tag strings (SPF/DMARC style).
func parseTags(rec string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(rec, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			out[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
		}
	}
	return out
}

// normalizeDomain strips a scheme, path, a mailto local-part, surrounding
// whitespace, and a trailing dot.
func normalizeDomain(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			s = u.Host
		}
	}
	if i := strings.LastIndexByte(s, '@'); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), ".")
	return strings.ToLower(s)
}
