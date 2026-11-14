// Package transcode is a local encode/decode swiss-army: base64, base64url,
// hex, URL percent-encoding, and JSON pretty/minify, both directions — plus a
// best-effort "what is this?" detector. It replaces pasting data into
// base64decode.org / jsonlint-style sites, keeping the (often sensitive) payload
// on the machine. Point-in-time; nothing is stored.
package transcode

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Apply runs one named operation on input.
func Apply(op, input string) (string, error) {
	switch op {
	case "base64-encode":
		return base64.StdEncoding.EncodeToString([]byte(input)), nil
	case "base64-decode":
		return decodeBase64(input)
	case "base64url-encode":
		return base64.RawURLEncoding.EncodeToString([]byte(input)), nil
	case "base64url-decode":
		return decodeBase64(input)
	case "hex-encode":
		return hex.EncodeToString([]byte(input)), nil
	case "hex-decode":
		b, err := hex.DecodeString(strings.Join(strings.Fields(input), ""))
		if err != nil {
			return "", fmt.Errorf("not valid hex: %w", err)
		}
		return string(b), nil
	case "url-encode":
		return url.QueryEscape(input), nil
	case "url-decode":
		s, err := url.QueryUnescape(input)
		if err != nil {
			return "", fmt.Errorf("not valid URL encoding: %w", err)
		}
		return s, nil
	case "json-pretty":
		var out bytes.Buffer
		if err := json.Indent(&out, []byte(input), "", "  "); err != nil {
			return "", fmt.Errorf("not valid JSON: %w", err)
		}
		return out.String(), nil
	case "json-minify":
		var out bytes.Buffer
		if err := json.Compact(&out, []byte(input)); err != nil {
			return "", fmt.Errorf("not valid JSON: %w", err)
		}
		return out.String(), nil
	}
	return "", fmt.Errorf("unknown operation %q", op)
}

// Ops lists the supported operation names, in a sensible UI order.
func Ops() []string {
	return []string{
		"base64-encode", "base64-decode",
		"base64url-encode", "base64url-decode",
		"hex-encode", "hex-decode",
		"url-encode", "url-decode",
		"json-pretty", "json-minify",
	}
}

// decodeBase64 is lenient: it accepts standard or URL alphabets, with or without
// padding.
func decodeBase64(s string) (string, error) {
	s = strings.TrimSpace(s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(strings.TrimRight(s, "=")); err == nil {
			return string(b), nil
		}
	}
	return "", fmt.Errorf("not valid base64")
}

// Guess is one plausible interpretation of some input.
type Guess struct {
	Label  string `json:"label"`
	Result string `json:"result"`
}

// Detect best-effort reports how the input might be decoded — base64/hex/URL to
// readable text, or JSON to pretty-printed — returning only interpretations that
// produce sensible output.
func Detect(input string) []Guess {
	in := strings.TrimSpace(input)
	var out []Guess
	if in == "" {
		return out
	}
	if b, err := decodeBase64(in); err == nil && b != in && printable(b) {
		out = append(out, Guess{"Base64 → text", b})
	}
	if h, err := hex.DecodeString(strings.Join(strings.Fields(in), "")); err == nil && len(h) > 0 && printable(string(h)) {
		out = append(out, Guess{"Hex → text", string(h)})
	}
	if u, err := url.QueryUnescape(in); err == nil && u != in {
		out = append(out, Guess{"URL-decoded", u})
	}
	if json.Valid([]byte(in)) {
		var buf bytes.Buffer
		if json.Indent(&buf, []byte(in), "", "  ") == nil && buf.String() != in {
			out = append(out, Guess{"JSON (pretty)", buf.String()})
		}
	}
	return out
}

// printable reports whether s is valid UTF-8 with no unexpected control bytes
// (so a base64/hex "decode" that produced binary garbage isn't offered as text).
func printable(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
