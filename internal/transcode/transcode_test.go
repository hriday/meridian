package transcode

import (
	"strings"
	"testing"
)

func TestRoundTrips(t *testing.T) {
	cases := []struct{ enc, dec, text string }{
		{"base64-encode", "base64-decode", "hello world"},
		{"base64url-encode", "base64url-decode", "sub?a=b&c=d"},
		{"hex-encode", "hex-decode", "Meridian"},
		{"url-encode", "url-decode", "a b&c=d/e"},
	}
	for _, c := range cases {
		e, err := Apply(c.enc, c.text)
		if err != nil {
			t.Fatalf("%s: %v", c.enc, err)
		}
		back, err := Apply(c.dec, e)
		if err != nil {
			t.Fatalf("%s: %v", c.dec, err)
		}
		if back != c.text {
			t.Errorf("%s/%s round-trip: got %q, want %q", c.enc, c.dec, back, c.text)
		}
	}
}

func TestBase64Known(t *testing.T) {
	out, _ := Apply("base64-encode", "Meridian")
	if out != "TWVyaWRpYW4=" {
		t.Errorf("base64 of Meridian = %q, want TWVyaWRpYW4=", out)
	}
	back, _ := Apply("base64-decode", "TWVyaWRpYW4=")
	if back != "Meridian" {
		t.Errorf("decode = %q", back)
	}
}

func TestJSONPrettyMinify(t *testing.T) {
	pretty, err := Apply("json-pretty", `{"b":2,"a":1}`)
	if err != nil || !strings.Contains(pretty, "\n") {
		t.Errorf("json-pretty: %q err=%v", pretty, err)
	}
	min, err := Apply("json-minify", "{\n  \"a\": 1\n}")
	if err != nil || min != `{"a":1}` {
		t.Errorf("json-minify = %q err=%v", min, err)
	}
	if _, err := Apply("json-pretty", "not json"); err == nil {
		t.Error("invalid JSON should error")
	}
}

func TestDetect(t *testing.T) {
	g := Detect("TWVyaWRpYW4=")
	found := false
	for _, x := range g {
		if x.Result == "Meridian" {
			found = true
		}
	}
	if !found {
		t.Errorf("should detect base64 → Meridian, got %+v", g)
	}

	// Binary garbage base64 should NOT be offered as text.
	for _, x := range Detect("////") {
		if strings.HasPrefix(x.Label, "Base64") {
			t.Errorf("non-printable base64 should not be offered as text: %q", x.Result)
		}
	}
}

func TestUnknownOp(t *testing.T) {
	if _, err := Apply("rot13", "x"); err == nil {
		t.Error("unknown op should error")
	}
}
