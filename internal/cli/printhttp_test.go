package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/httpcheck"
)

func TestPrintHTTPResult(t *testing.T) {
	r := httpcheck.Result{
		FinalURL: "https://example.com/", Status: 200, StatusText: "OK", Proto: "HTTP/2.0",
		Server: "nginx", ContentType: "text/html", TLSVersion: "TLS 1.3",
		Redirects: []httpcheck.Hop{{Status: 301, URL: "http://example.com/"}},
		Timing:    httpcheck.Timing{DNSMillis: 5, ConnectMillis: 10, TLSMillis: 20, TTFBMillis: 40, TotalMillis: 45},
	}
	var buf bytes.Buffer
	printHTTPResult(&buf, r)
	out := buf.String()
	for _, want := range []string{"200", "OK", "HTTP/2.0", "nginx", "text/html", "TLS 1.3", "301", "http://example.com/", "40"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRunHTTPMissingArg(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runHTTP(nil, &out, &errb); code != 2 {
		t.Errorf("runHTTP with no args: got exit %d, want 2", code)
	}
}
