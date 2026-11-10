package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/macvendor"
)

func TestPrintMACVendorFound(t *testing.T) {
	var buf bytes.Buffer
	printMACVendorReport(&buf, macvendor.Result{OUI: "FC:FB:FB", Vendor: "Apple, Inc.", Found: true})
	out := buf.String()
	if !strings.Contains(out, "FC:FB:FB") || !strings.Contains(out, "Apple, Inc.") {
		t.Errorf("found output missing OUI or vendor:\n%s", out)
	}
}

func TestPrintMACVendorNotFound(t *testing.T) {
	var buf bytes.Buffer
	printMACVendorReport(&buf, macvendor.Result{OUI: "02:00:00", Found: false, LocallyAdministered: true})
	out := strings.ToLower(buf.String())
	if !strings.Contains(out, "not") || !strings.Contains(out, "registered") {
		t.Errorf("not-found output should say it's not registered:\n%s", buf.String())
	}
	if !strings.Contains(out, "local") && !strings.Contains(out, "random") {
		t.Errorf("locally-administered hint missing:\n%s", buf.String())
	}
}

func TestRunMACVendorMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runMACVendor(nil, &out, &errb); code != 2 {
		t.Errorf("runMACVendor with no args: got exit %d, want 2", code)
	}
}
