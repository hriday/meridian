package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/svcstatus"
)

func TestPrintServiceStatus(t *testing.T) {
	var buf bytes.Buffer
	printServiceStatus(&buf, svcstatus.Status{
		Service: "GitHub", Description: "Partial System Outage", Operational: false,
		Impacted: []svcstatus.Component{{Name: "Actions", Status: "major_outage"}},
		Incidents: []svcstatus.Incident{{Name: "Actions degraded", Status: "investigating", Impact: "major", Shortlink: "https://stspg.io/x"}},
	})
	out := buf.String()
	for _, want := range []string{"OUTAGE", "GitHub", "Partial System Outage", "Actions", "major outage", "Actions degraded", "stspg.io"} {
		if !strings.Contains(out, want) {
			t.Errorf("printServiceStatus missing %q\n%s", want, out)
		}
	}
}

func TestPrintServiceStatusOK(t *testing.T) {
	var buf bytes.Buffer
	printServiceStatus(&buf, svcstatus.Status{Service: "Cloudflare", Description: "All Systems Operational", Operational: true})
	if !strings.Contains(buf.String(), "[OK]") {
		t.Errorf("operational service should read OK: %q", buf.String())
	}
}

func TestRunServiceStatusMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runServiceStatus(nil, &o, &e); code != 2 {
		t.Errorf("status with no args: got exit %d, want 2", code)
	}
}
