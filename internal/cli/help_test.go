package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpIsGroupedAndComplete(t *testing.T) {
	var out, errb bytes.Buffer
	if code := Run([]string{"help"}, &out, &errb); code != 0 {
		t.Fatalf("help exit %d", code)
	}
	s := out.String()

	for _, group := range []string{"Internet", "Addressing", "Local network", "SNMP & monitoring", "Collectors", "Security", "Config insight"} {
		if !strings.Contains(s, group) {
			t.Errorf("help is missing the %q group heading", group)
		}
	}
	// A spread of commands across groups, including the newest tools.
	for _, cmd := range []string{"lookup", "cipherscan", "emailauth", "certexpiry", "ptrsweep", "snmpbrute", "syslog", "threatintel", "cfgdiff"} {
		if !strings.Contains(s, cmd) {
			t.Errorf("help doesn't document the %q command", cmd)
		}
	}
}

func TestUnknownCommandShowsShortUsage(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"definitely-not-a-command"}, &out, &errb)
	if code != 2 {
		t.Errorf("unknown command exit = %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "meridian-cli help") {
		t.Errorf("error should point at 'meridian-cli help': %q", errb.String())
	}
	// The short usage shouldn't dump every command line.
	if strings.Contains(errb.String(), "SNMP & monitoring") {
		t.Error("errors should show the short usage, not the full grouped help")
	}
}
