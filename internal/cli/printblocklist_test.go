package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dnsbl"
)

func TestPrintBlocklistReportListed(t *testing.T) {
	r := dnsbl.Report{
		IP:          "1.2.3.4",
		Resolver:    "192.168.1.1:53",
		Checked:     3,
		Listed:      1,
		Unreachable: 1,
		Verdict:     "Listed on 1 blocklist(s)",
		Summary:     "Listed on Spamhaus ZEN. Incomplete: 1 of 3 list(s) could not be reached, so this is not a full all-clear.",
		Listings: []dnsbl.Listing{
			{Zone: "zen.spamhaus.org", Name: "Spamhaus ZEN", Status: dnsbl.StatusListed, Codes: []string{"127.0.0.2"}, Reason: "see https://example.com/remove"},
			{Zone: "bl.spamcop.net", Name: "SpamCop", Status: dnsbl.StatusClean},
			{Zone: "dnsbl.dronebl.org", Name: "DroneBL", Status: dnsbl.StatusBlocked, Note: "use your own resolver"},
		},
	}
	var buf bytes.Buffer
	printBlocklistReport(&buf, r)
	out := buf.String()

	for _, want := range []string{
		"Listed on 1 blocklist(s)", // verdict
		"192.168.1.1:53",           // resolver
		"Spamhaus ZEN",             // listed list name
		"127.0.0.2",                // code
		"example.com/remove",       // reason
		"SpamCop",                  // clean list
		"DroneBL",                  // blocked list
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	if !strings.Contains(strings.ToLower(out), "incomplete") {
		t.Errorf("output should surface the incomplete check:\n%s", out)
	}
}

func TestRunBlocklistMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runBlocklist(nil, &out, &errb); code != 2 {
		t.Errorf("runBlocklist with no args: got exit %d, want 2", code)
	}
}
