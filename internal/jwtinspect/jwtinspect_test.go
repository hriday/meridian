package jwtinspect

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
	"time"
)

func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
func b64b(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func hs256Token(secret, header, payload string) string {
	input := b64(header) + "." + b64(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return input + "." + b64b(mac.Sum(nil))
}

func pubPEM(t *testing.T, pub any) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func TestDecodeClaims(t *testing.T) {
	now := time.Now()
	payload := fmt.Sprintf(`{"sub":"1234567890","iss":"meridian","aud":["a","b"],"exp":%d,"iat":%d}`,
		now.Add(time.Hour).Unix(), now.Unix())
	tok := hs256Token("secret", `{"alg":"HS256","typ":"JWT","kid":"k1"}`, payload)

	d, err := Decode(tok)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if d.Alg != "HS256" || d.Typ != "JWT" || d.Kid != "k1" {
		t.Errorf("header: alg=%q typ=%q kid=%q", d.Alg, d.Typ, d.Kid)
	}
	if d.Subject != "1234567890" || d.Issuer != "meridian" {
		t.Errorf("claims: sub=%q iss=%q", d.Subject, d.Issuer)
	}
	if len(d.Audience) != 2 || d.Audience[1] != "b" {
		t.Errorf("aud = %v, want [a b]", d.Audience)
	}
	if d.Expired || d.ExpiresAt == nil {
		t.Errorf("should not be expired; expiresAt=%v", d.ExpiresAt)
	}
	if !strings.Contains(d.PayloadJSON, "meridian") {
		t.Errorf("payload JSON not pretty-printed: %q", d.PayloadJSON)
	}
}

func TestDecodeExpired(t *testing.T) {
	payload := fmt.Sprintf(`{"sub":"x","exp":%d}`, time.Now().Add(-time.Hour).Unix())
	tok := hs256Token("s", `{"alg":"HS256"}`, payload)
	d, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Expired {
		t.Error("token with a past exp should be flagged expired")
	}
	if len(d.Warnings) == 0 {
		t.Error("expired token should carry a warning")
	}
}

func TestDecodeAlgNoneWarns(t *testing.T) {
	tok := b64(`{"alg":"none"}`) + "." + b64(`{"sub":"x"}`) + "."
	d, err := Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(d.Warnings, " ")
	if !strings.Contains(joined, "none") {
		t.Errorf("alg:none should warn, got %v", d.Warnings)
	}
}

func TestDecodeRejectsJWE(t *testing.T) {
	jwe := "a.b.c.d.e"
	if _, err := Decode(jwe); err == nil {
		t.Error("a 5-part JWE should be rejected with a clear error")
	}
}

func TestVerifyHS256(t *testing.T) {
	tok := hs256Token("topsecret", `{"alg":"HS256"}`, `{"sub":"x"}`)

	ok, err := Verify(tok, "topsecret")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok.Valid {
		t.Errorf("correct secret should verify: %s", ok.Reason)
	}

	bad, _ := Verify(tok, "wrongsecret")
	if bad.Valid {
		t.Error("wrong secret must not verify")
	}
}

func TestVerifyNoneIsInvalid(t *testing.T) {
	tok := b64(`{"alg":"none"}`) + "." + b64(`{"sub":"x"}`) + "."
	r, err := Verify(tok, "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Valid {
		t.Error("alg:none must never be reported valid")
	}
}

func TestVerifyRS256(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	input := b64(`{"alg":"RS256"}`) + "." + b64(`{"sub":"x"}`)
	sum := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, 5 /*crypto.SHA256*/, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	tok := input + "." + b64b(sig)

	r, err := Verify(tok, pubPEM(t, &key.PublicKey))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !r.Valid {
		t.Errorf("valid RS256 signature should verify: %s", r.Reason)
	}
	// Tamper with the payload.
	bad := b64(`{"alg":"RS256"}`) + "." + b64(`{"sub":"admin"}`) + "." + b64b(sig)
	if br, _ := Verify(bad, pubPEM(t, &key.PublicKey)); br.Valid {
		t.Error("tampered token must not verify")
	}
}

func TestVerifyES256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input := b64(`{"alg":"ES256"}`) + "." + b64(`{"sub":"x"}`)
	sum := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, key, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	tok := input + "." + b64b(sig)

	res, err := Verify(tok, pubPEM(t, &key.PublicKey))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !res.Valid {
		t.Errorf("valid ES256 signature should verify: %s", res.Reason)
	}
}
