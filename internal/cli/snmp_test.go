package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintDevHealth(t *testing.T) {
	var buf bytes.Buffer
	printDevHealth(&buf, devHealthOut{
		Device: "10.0.0.1", Name: "core-sw", Descr: "Cisco IOS XE",
		UptimeSecs: 90061, CPUPct: 23, MemPct: 61, HasCPU: true, HasMem: true,
		Interfaces: 48, IfUp: 40,
	})
	out := buf.String()
	for _, want := range []string{"10.0.0.1", "core-sw", "Cisco IOS XE", "23%", "61%", "48 (40 up)"} {
		if !strings.Contains(out, want) {
			t.Errorf("printDevHealth missing %q\n%s", want, out)
		}
	}
}

func TestPrintDevHealthNA(t *testing.T) {
	var buf bytes.Buffer
	printDevHealth(&buf, devHealthOut{Device: "d", Name: "n", HasCPU: false, HasMem: false})
	if !strings.Contains(buf.String(), "n/a") {
		t.Errorf("missing CPU/mem should read n/a:\n%s", buf.String())
	}
}

func TestScrubCommunities(t *testing.T) {
	got := scrubCommunities("snmp connect failed with s3cr3t and public", []string{"s3cr3t", "public"})
	if strings.Contains(got, "s3cr3t") || strings.Contains(got, "public") {
		t.Errorf("community leaked through scrub: %q", got)
	}
	if !strings.Contains(got, "***") {
		t.Errorf("scrub should mask: %q", got)
	}
}

func TestSNMPBruteRequiresAuthorization(t *testing.T) {
	var out, errb bytes.Buffer
	// A device given but no --authorized flag → usage error, nothing attempted.
	if code := runSNMPBrute([]string{"10.0.0.1"}, &out, &errb); code != 2 {
		t.Errorf("snmpbrute without --authorized: got exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "authorized") {
		t.Errorf("should explain the authorization gate: %q", errb.String())
	}
}

func TestCommunityListDefaults(t *testing.T) {
	got, err := communityList("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0] != "public" {
		t.Errorf("default list should start with public, got %v", got)
	}
}

func TestCommunityListCSV(t *testing.T) {
	got, err := communityList("alpha, beta ,, gamma", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "beta", "gamma"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("CSV parse: got %v, want %v", got, want)
	}
}

func TestSNMPCommandsMissingArg(t *testing.T) {
	cases := map[string]func([]string) int{
		"snmpsweep": func(a []string) int { var o, e bytes.Buffer; return runSNMPSweep(a, &o, &e) },
		"devhealth": func(a []string) int { var o, e bytes.Buffer; return runDevHealth(a, &o, &e) },
		"swport":    func(a []string) int { var o, e bytes.Buffer; return runSwitchPort(a, &o, &e) },
		"mib":       func(a []string) int { var o, e bytes.Buffer; return runMIB(a, &o, &e) },
		"snmpbrute": func(a []string) int { var o, e bytes.Buffer; return runSNMPBrute(a, &o, &e) },
	}
	for name, fn := range cases {
		if code := fn(nil); code != 2 {
			t.Errorf("%s with no args: got exit %d, want 2", name, code)
		}
	}
}
