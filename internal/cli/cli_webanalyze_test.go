package cli

import (
	"bytes"
	"strings"
	"testing"
)

const nginxCLISample = `http {
    upstream api { server 10.0.0.1:8080; }
    server {
        listen 443 ssl;
        ssl_certificate /etc/ssl/x.crt;
        server_name site.com;
        location /api { proxy_pass http://api; }
    }
}`

func TestRunWebAnalyzeStdinJSON(t *testing.T) {
	old := webStdin
	webStdin = strings.NewReader(nginxCLISample)
	defer func() { webStdin = old }()

	var out, errb bytes.Buffer
	if code := runWebAnalyze([]string{"--json"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errb.String())
	}
	s := out.String()
	for _, want := range []string{`"servers"`, `"site.com"`, `"upstreams"`, `"api"`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %q\n%s", want, s)
		}
	}
}

func TestRunWebAnalyzeTextSummary(t *testing.T) {
	old := webStdin
	webStdin = strings.NewReader(nginxCLISample)
	defer func() { webStdin = old }()

	var out, errb bytes.Buffer
	if code := runWebAnalyze(nil, &out, &errb); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "site.com") || !strings.Contains(s, "api") {
		t.Errorf("summary missing content\n%s", s)
	}
}

func TestRunWebAnalyzeEmpty(t *testing.T) {
	old := webStdin
	webStdin = strings.NewReader("")
	defer func() { webStdin = old }()
	var out, errb bytes.Buffer
	if code := runWebAnalyze(nil, &out, &errb); code != 1 {
		t.Errorf("empty: got %d want 1", code)
	}
}
