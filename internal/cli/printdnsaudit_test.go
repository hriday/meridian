package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dnsaudit"
)

func TestPrintDNSAuditReport(t *testing.T) {
	r := dnsaudit.Report{
		Domain:   "example.com",
		Resolver: "192.168.1.1:53",
		OKs:      3, Warns: 1, Missing: 1,
		Summary: "DMARC needs attention — No DMARC record.",
		Checks: []dnsaudit.Check{
			{Name: "Nameservers", Status: dnsaudit.StatusOK, Note: "Served by 2 nameserver(s).", Records: []string{"ns1.example.com."}},
			{Name: "SPF", Status: dnsaudit.StatusWarn, Note: "SPF ends in ?all (neutral).", Records: []string{"v=spf1 ?all"}},
			{Name: "DMARC", Status: dnsaudit.StatusMissing, Note: "No DMARC record."},
		},
	}
	var buf bytes.Buffer
	printDNSAuditReport(&buf, r)
	out := buf.String()
	for _, want := range []string{
		"DMARC needs attention", // summary
		"192.168.1.1:53",        // resolver
		"Nameservers",
		"SPF",
		"ns1.example.com.", // a record
		"v=spf1 ?all",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRunDNSAuditMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runDNSAudit(nil, &out, &errb); code != 2 {
		t.Errorf("runDNSAudit with no args: got exit %d, want 2", code)
	}
}
