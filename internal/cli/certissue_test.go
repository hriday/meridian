package cli

import (
	"bytes"
	"testing"
)

func TestRunCertIssueMissingDomain(t *testing.T) {
	var o, e bytes.Buffer
	if code := runCertIssue(nil, &o, &e); code != 2 {
		t.Errorf("certissue with no domain: got exit %d, want 2", code)
	}
}
