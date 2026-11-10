package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/lookup"
)

func TestPrintDomainReportWhoisFields(t *testing.T) {
	r := lookup.DomainReport{
		Input:      "example.com",
		Registrar:  "Example Registrar, Inc.",
		Created:    "2009-02-17T22:07:54Z",
		Updated:    "2024-05-01T00:00:00Z",
		Expires:    "2026-02-17T00:00:00Z",
		DNSSEC:     "signed",
		Statuses:   []string{"clientTransferProhibited", "clientDeleteProhibited"},
		AbuseEmail: "abuse@registrar.example",
	}
	var buf bytes.Buffer
	printDomainReport(&buf, r)
	out := buf.String()
	for _, want := range []string{
		"clientTransferProhibited", // status code
		"2024-05-01",               // updated date
		"abuse@registrar.example",  // abuse contact
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestPrintDomainReportOmitsEmptyWhoisRows(t *testing.T) {
	r := lookup.DomainReport{Input: "example.com", Registrar: "R", Created: "c", Expires: "e", DNSSEC: "unsigned"}
	var buf bytes.Buffer
	printDomainReport(&buf, r)
	out := buf.String()
	for _, absent := range []string{"Status", "Updated", "Abuse"} {
		if strings.Contains(out, absent) {
			t.Errorf("output should omit the %q row when empty\n---\n%s", absent, out)
		}
	}
}
