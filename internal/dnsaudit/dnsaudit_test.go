package dnsaudit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSPFVerdict(t *testing.T) {
	cases := []struct {
		name    string
		in      []string
		want    Status
		noteSub string // substring expected in the note (optional)
		rec     bool   // expect the matched record returned
	}{
		{"strict", []string{"v=spf1 -all"}, StatusOK, "", true},
		{"soft", []string{"v=spf1 include:_spf.example.com ~all"}, StatusOK, "soft", true},
		{"neutral", []string{"v=spf1 ?all"}, StatusWarn, "neutral", true},
		{"passall", []string{"v=spf1 +all"}, StatusWarn, "any", true},
		{"noall", []string{"v=spf1 include:x"}, StatusWarn, "", true},
		{"multiple", []string{"v=spf1 -all", "v=spf1 ~all"}, StatusWarn, "more than one", true},
		{"notspf", []string{"not spf"}, StatusMissing, "", false},
		{"none", nil, StatusMissing, "", false},
	}
	for _, c := range cases {
		st, note, rec := spfVerdict(c.in)
		require.Equal(t, c.want, st, c.name)
		if c.noteSub != "" {
			require.Contains(t, strings.ToLower(note), c.noteSub, c.name)
		}
		if c.rec {
			require.NotEmpty(t, rec, c.name)
		}
	}
}

func TestDMARCVerdict(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want Status
	}{
		{"reject", []string{"v=DMARC1; p=reject; rua=mailto:a@b.c"}, StatusOK},
		{"quarantine", []string{"v=DMARC1; p=quarantine"}, StatusOK},
		{"none", []string{"v=DMARC1; p=none"}, StatusWarn},
		{"no-p", []string{"v=DMARC1; rua=mailto:x@y.z"}, StatusWarn},
		{"notdmarc", []string{"something else"}, StatusMissing},
		{"empty", nil, StatusMissing},
	}
	for _, c := range cases {
		st, _, _ := dmarcVerdict(c.in)
		require.Equal(t, c.want, st, c.name)
	}
}
