package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/certexpiry"
	"github.com/hriday/meridian/internal/ctlog"
	"github.com/hriday/meridian/internal/cve"
	"github.com/hriday/meridian/internal/emailauth"
	"github.com/hriday/meridian/internal/hashid"
	"github.com/hriday/meridian/internal/secheaders"
	"github.com/hriday/meridian/internal/threatintel"
)

// ---- cve -------------------------------------------------------------------

func runCVE(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) == 0 {
		fmt.Fprintf(stderr, "cve needs a CVE id or a keyword\n%s", usage)
		return 2
	}
	q := strings.Join(pos, " ")
	client := &http.Client{Timeout: 20 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if cve.IsID(q) {
		c, err := cve.Lookup(ctx, client, q)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			return writeJSON(stdout, stderr, c)
		}
		printCVE(stdout, c)
		return 0
	}
	list, err := cve.Search(ctx, client, q)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, list)
	}
	printCVEList(stdout, list)
	return 0
}

func printCVE(stdout io.Writer, c cve.CVE) {
	reportRow(stdout, "CVE", c.ID)
	reportRow(stdout, "Published", c.Published)
	sev := c.Severity
	if c.CVSS > 0 {
		sev = fmt.Sprintf("%s (CVSS %.1f)", c.Severity, c.CVSS)
	}
	reportRow(stdout, "Severity", sev)
	if c.Vector != "" {
		reportRow(stdout, "Vector", c.Vector)
	}
	if len(c.CWE) > 0 {
		reportRow(stdout, "CWE", strings.Join(c.CWE, ", "))
	}
	if c.KEV {
		reportRow(stdout, "KEV", "YES — CISA Known Exploited Vulnerability")
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, c.Description)
	if len(c.References) > 0 {
		fmt.Fprintln(stdout, "\nReferences:")
		for _, r := range c.References {
			fmt.Fprintf(stdout, "  %s\n", r)
		}
	}
}

func printCVEList(stdout io.Writer, list []cve.CVE) {
	if len(list) == 0 {
		fmt.Fprintln(stdout, "no matching CVEs")
		return
	}
	for _, c := range list {
		sev := c.Severity
		if sev == "" {
			sev = "—"
		}
		kev := ""
		if c.KEV {
			kev = " [KEV]"
		}
		fmt.Fprintf(stdout, "%-18s %-9s%s %s\n", c.ID, sev, kev, firstLine(c.Description, 90))
	}
}

// ---- hashid ----------------------------------------------------------------

func runHashID(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hashid", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "hashid needs exactly one hash\n%s", usage)
		return 2
	}
	cands := hashid.Identify(pos[0])
	if *asJSON {
		return writeJSON(stdout, stderr, cands)
	}
	printHashID(stdout, cands)
	return 0
}

func printHashID(stdout io.Writer, cands []hashid.Candidate) {
	for _, c := range cands {
		mode := "no hashcat mode"
		if c.Mode >= 0 {
			mode = fmt.Sprintf("hashcat -m %d", c.Mode)
		}
		line := fmt.Sprintf("%-40s %s", c.Name, mode)
		if c.Note != "" {
			line += "  (" + c.Note + ")"
		}
		fmt.Fprintln(stdout, line)
	}
}

// ---- threatintel -----------------------------------------------------------

func runThreatIntel(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("threatintel", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) == 0 {
		fmt.Fprintf(stderr, "threatintel needs at least one IP\n%s", usage)
		return 2
	}
	client := &http.Client{Timeout: 25 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hits, err := threatintel.Check(ctx, client, pos)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, hits)
	}
	printThreatIntel(stdout, hits)
	return 0
}

func printThreatIntel(stdout io.Writer, hits []threatintel.Hit) {
	for _, h := range hits {
		if h.Listed {
			fmt.Fprintf(stdout, "%-18s BOTNET C2  %s  (%s, %s, last online %s)\n",
				h.IP, h.Malware, h.Status, h.Country, h.LastOnline)
		} else {
			fmt.Fprintf(stdout, "%-18s clean\n", h.IP)
		}
	}
}

// ---- certexpiry ------------------------------------------------------------

func runCertExpiry(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("certexpiry", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) == 0 {
		fmt.Fprintf(stderr, "certexpiry needs at least one host\n%s", usage)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res := certexpiry.Check(ctx, certexpiry.Dial, pos, time.Now())
	if *asJSON {
		return writeJSON(stdout, stderr, res)
	}
	printCertExpiry(stdout, res)
	return 0
}

func printCertExpiry(stdout io.Writer, res []certexpiry.Result) {
	for _, r := range res {
		if !r.OK {
			fmt.Fprintf(stdout, "%-32s unreachable (%s)\n", r.Host, r.Error)
			continue
		}
		var state string
		switch {
		case r.NotYet:
			state = "not valid yet"
		case r.Expired:
			state = fmt.Sprintf("EXPIRED %dd ago", -r.DaysLeft)
		default:
			state = fmt.Sprintf("%d days left", r.DaysLeft)
		}
		fmt.Fprintf(stdout, "%-32s %-18s expires %s  (%s)\n", r.Host, state, r.NotAfter, r.Issuer)
	}
}

// ---- sechdr ----------------------------------------------------------------

func runSecHeaders(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sechdr", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "sechdr needs exactly one URL\n%s", usage)
		return 2
	}
	client := &http.Client{Timeout: 15 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rep, err := secheaders.Scan(ctx, client, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printSecHeaders(stdout, rep)
	return 0
}

func printSecHeaders(stdout io.Writer, r secheaders.Report) {
	reportRow(stdout, "URL", r.FinalURL)
	reportRow(stdout, "Score", fmt.Sprintf("%d/100  grade %s", r.Score, r.Grade))
	fmt.Fprintln(stdout)
	for _, h := range r.Headers {
		mark := "✓"
		if !h.Present {
			mark = "✗"
		}
		line := fmt.Sprintf("%s [%s] %-26s", mark, strings.ToUpper(emptyDash(h.Grade)), h.Name)
		if h.OWASP != "" {
			line += "  " + h.OWASP
		}
		fmt.Fprintln(stdout, line)
		if h.Note != "" {
			fmt.Fprintf(stdout, "      %s\n", h.Note)
		}
	}
}

// ---- subdomains ------------------------------------------------------------

func runSubdomains(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("subdomains", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "subdomains needs exactly one domain\n%s", usage)
		return 2
	}
	client := &http.Client{Timeout: 45 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	subs, err := ctlog.Subdomains(ctx, client, pos[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *asJSON {
		return writeJSON(stdout, stderr, subs)
	}
	printSubdomains(stdout, subs)
	return 0
}

func printSubdomains(stdout io.Writer, subs []ctlog.Subdomain) {
	fmt.Fprintf(stdout, "%d subdomains from Certificate Transparency\n\n", len(subs))
	for _, s := range subs {
		name := s.Name
		if s.Wildcard {
			name = "*." + name
		}
		fmt.Fprintf(stdout, "%-40s %3d certs  (%s → %s)\n", name, s.Certs, s.FirstSeen, s.LastSeen)
	}
}

// ---- emailauth -------------------------------------------------------------

func runEmailAuth(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("emailauth", flag.ContinueOnError)
	fs.SetOutput(stderr)
	selectors := fs.String("selectors", "", "comma-separated DKIM selectors (default: a built-in set)")
	asJSON := fs.Bool("json", false, "output JSON")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintf(stderr, "emailauth needs exactly one domain\n%s", usage)
		return 2
	}
	var sels []string
	for _, s := range strings.Split(*selectors, ",") {
		if s = strings.TrimSpace(s); s != "" {
			sels = append(sels, s)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	lookup := func(ctx context.Context, name string) ([]string, error) {
		return net.DefaultResolver.LookupTXT(ctx, name)
	}
	rep := emailauth.Check(ctx, lookup, pos[0], sels)
	if *asJSON {
		return writeJSON(stdout, stderr, rep)
	}
	printEmailAuth(stdout, rep)
	return 0
}

func printEmailAuth(stdout io.Writer, r emailauth.Report) {
	reportRow(stdout, "Domain", r.Domain)
	reportRow(stdout, "Summary", r.Summary)
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "SPF   [%-4s] %s\n", strings.ToUpper(string(r.SPF.Status)), r.SPF.Note)
	if r.SPF.Record != "" {
		fmt.Fprintf(stdout, "        %s\n", r.SPF.Record)
	}
	fmt.Fprintf(stdout, "DMARC [%-4s] %s\n", strings.ToUpper(string(r.DMARC.Status)), r.DMARC.Note)
	fmt.Fprintf(stdout, "DKIM  [%-4s] ", upperBool(r.DKIMFound))
	if r.DKIMFound {
		var found []string
		for _, d := range r.DKIM {
			if d.Found {
				found = append(found, d.Selector)
			}
		}
		fmt.Fprintf(stdout, "selectors with a key: %s\n", strings.Join(found, ", "))
	} else {
		fmt.Fprintf(stdout, "no key found among %d probed selectors\n", len(r.DKIM))
	}
}

func upperBool(b bool) string {
	if b {
		return "PASS"
	}
	return "WARN"
}

// ---- small shared formatting helpers ---------------------------------------

func firstLine(s string, max int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if max > 0 && len(s) > max {
		s = s[:max-1] + "…"
	}
	return s
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
