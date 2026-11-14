// Package jwtinspect decodes and verifies JSON Web Tokens entirely locally —
// the token (a bearer credential) never leaves the machine, unlike pasting it
// into a website. It pretty-prints the header and payload, interprets the
// standard claims (expiry, issuer, …), flags footguns like "alg: none", and
// verifies the signature against a secret (HS*) or a PEM public key (RS*, PS*,
// ES*, EdDSA) — all with the standard library, no JWT dependency.
package jwtinspect

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Decoded is a JWT taken apart.
type Decoded struct {
	HeaderJSON  string         `json:"headerJSON"`  // pretty-printed, original key order
	PayloadJSON string         `json:"payloadJSON"` // pretty-printed, original key order
	Alg         string         `json:"alg"`
	Typ         string         `json:"typ,omitempty"`
	Kid         string         `json:"kid,omitempty"`
	Issuer      string         `json:"issuer,omitempty"`
	Subject     string         `json:"subject,omitempty"`
	Audience    []string       `json:"audience,omitempty"`
	JWTID       string         `json:"jwtID,omitempty"`
	IssuedAt    *time.Time     `json:"issuedAt,omitempty"`
	NotBefore   *time.Time     `json:"notBefore,omitempty"`
	ExpiresAt   *time.Time     `json:"expiresAt,omitempty"`
	Expired     bool           `json:"expired"`
	NotYetValid bool           `json:"notYetValid"`
	Claims      map[string]any `json:"-"`
	Warnings    []string       `json:"warnings,omitempty"`
}

// Decode splits a compact JWT and interprets it. It does not verify the
// signature (that's Verify); it reports what the token claims.
func Decode(token string) (Decoded, error) {
	return decodeAt(token, time.Now())
}

func decodeAt(token string, now time.Time) (Decoded, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) == 5 {
		return Decoded{}, errors.New("this looks like a JWE (encrypted, 5 parts) — it can't be decoded without the decryption key")
	}
	if len(parts) != 3 && len(parts) != 2 {
		return Decoded{}, errors.New("not a JWT: expected 3 dot-separated parts (header.payload.signature)")
	}

	headerBytes, err := b64urlDecode(parts[0])
	if err != nil {
		return Decoded{}, fmt.Errorf("header isn't valid base64url: %w", err)
	}
	payloadBytes, err := b64urlDecode(parts[1])
	if err != nil {
		return Decoded{}, fmt.Errorf("payload isn't valid base64url: %w", err)
	}

	var d Decoded
	if d.HeaderJSON, err = prettyJSON(headerBytes); err != nil {
		return Decoded{}, fmt.Errorf("header isn't valid JSON: %w", err)
	}
	if d.PayloadJSON, err = prettyJSON(payloadBytes); err != nil {
		return Decoded{}, fmt.Errorf("payload isn't valid JSON: %w", err)
	}

	var hdr struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
		Kid string `json:"kid"`
	}
	_ = json.Unmarshal(headerBytes, &hdr)
	d.Alg, d.Typ, d.Kid = hdr.Alg, hdr.Typ, hdr.Kid

	claims := map[string]any{}
	_ = json.Unmarshal(payloadBytes, &claims)
	d.Claims = claims
	d.Issuer = claimString(claims, "iss")
	d.Subject = claimString(claims, "sub")
	d.JWTID = claimString(claims, "jti")
	d.Audience = claimStrings(claims, "aud")
	d.IssuedAt = claimTime(claims, "iat")
	d.NotBefore = claimTime(claims, "nbf")
	d.ExpiresAt = claimTime(claims, "exp")

	if d.ExpiresAt != nil && now.After(*d.ExpiresAt) {
		d.Expired = true
		d.Warnings = append(d.Warnings, "token is expired (exp is in the past)")
	}
	if d.NotBefore != nil && now.Before(*d.NotBefore) {
		d.NotYetValid = true
		d.Warnings = append(d.Warnings, "token is not valid yet (nbf is in the future)")
	}
	if strings.EqualFold(d.Alg, "none") || len(parts) == 2 || parts[2] == "" {
		d.Warnings = append(d.Warnings, "alg is \"none\": the token is UNSIGNED and can be forged by anyone — never trust it")
	}
	return d, nil
}

// VerifyResult is the outcome of a signature check.
type VerifyResult struct {
	Algorithm string `json:"algorithm"`
	Valid     bool   `json:"valid"`
	Reason    string `json:"reason"`
}

// Verify checks the signature. For HS* algorithms key is the shared secret; for
// RS*/PS*/ES*/EdDSA it is a PEM-encoded public key (or certificate).
func Verify(token, key string) (VerifyResult, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return VerifyResult{}, errors.New("need a signed JWT with 3 parts to verify")
	}
	headerBytes, err := b64urlDecode(parts[0])
	if err != nil {
		return VerifyResult{}, fmt.Errorf("header isn't valid base64url: %w", err)
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &hdr); err != nil {
		return VerifyResult{}, fmt.Errorf("header isn't valid JSON: %w", err)
	}
	alg := strings.ToUpper(hdr.Alg)
	sig, err := b64urlDecode(parts[2])
	if err != nil {
		return VerifyResult{}, fmt.Errorf("signature isn't valid base64url: %w", err)
	}
	signingInput := []byte(parts[0] + "." + parts[1])
	res := VerifyResult{Algorithm: hdr.Alg}

	if alg == "NONE" {
		res.Reason = "token is unsigned (alg: none) — nothing to verify, and it must not be trusted"
		return res, nil
	}
	if strings.TrimSpace(key) == "" {
		return VerifyResult{}, errors.New("provide a secret (HS*) or a PEM public key (RS*/ES*/PS*/EdDSA) to verify")
	}

	hash, ok := hashFor(alg)
	if !ok && !strings.HasPrefix(alg, "ED") {
		return VerifyResult{}, fmt.Errorf("unsupported algorithm %q", hdr.Alg)
	}

	switch {
	case strings.HasPrefix(alg, "HS"):
		mac := hmac.New(hash.New, []byte(key))
		mac.Write(signingInput)
		res.Valid = hmac.Equal(sig, mac.Sum(nil))
	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "PS"),
		strings.HasPrefix(alg, "ES"), strings.HasPrefix(alg, "ED"):
		pub, err := parsePublicKey(key)
		if err != nil {
			return VerifyResult{}, err
		}
		res.Valid, err = verifyAsym(alg, hash, pub, signingInput, sig)
		if err != nil {
			return VerifyResult{}, err
		}
	default:
		return VerifyResult{}, fmt.Errorf("unsupported algorithm %q", hdr.Alg)
	}

	if res.Valid {
		res.Reason = "signature is valid"
	} else {
		res.Reason = "signature does NOT match — wrong key, or the token was tampered with"
	}
	return res, nil
}

func verifyAsym(alg string, hash crypto.Hash, pub any, signingInput, sig []byte) (bool, error) {
	switch {
	case strings.HasPrefix(alg, "RS"):
		k, ok := pub.(*rsa.PublicKey)
		if !ok {
			return false, errors.New("algorithm is RSA but the key isn't an RSA public key")
		}
		sum := digest(hash, signingInput)
		return rsa.VerifyPKCS1v15(k, hash, sum, sig) == nil, nil
	case strings.HasPrefix(alg, "PS"):
		k, ok := pub.(*rsa.PublicKey)
		if !ok {
			return false, errors.New("algorithm is RSA-PSS but the key isn't an RSA public key")
		}
		sum := digest(hash, signingInput)
		return rsa.VerifyPSS(k, hash, sum, sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}) == nil, nil
	case strings.HasPrefix(alg, "ES"):
		k, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return false, errors.New("algorithm is ECDSA but the key isn't an EC public key")
		}
		n := (k.Curve.Params().BitSize + 7) / 8
		if len(sig) != 2*n {
			return false, nil // malformed signature
		}
		r := new(big.Int).SetBytes(sig[:n])
		s := new(big.Int).SetBytes(sig[n:])
		sum := digest(hash, signingInput)
		return ecdsa.Verify(k, sum, r, s), nil
	case strings.HasPrefix(alg, "ED"):
		k, ok := pub.(ed25519.PublicKey)
		if !ok {
			return false, errors.New("algorithm is EdDSA but the key isn't an Ed25519 public key")
		}
		return ed25519.Verify(k, signingInput, sig), nil
	}
	return false, fmt.Errorf("unsupported algorithm")
}

// parsePublicKey accepts a PEM public key (PKIX) or certificate.
func parsePublicKey(key string) (any, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, errors.New("couldn't read a PEM block — paste a public key (-----BEGIN PUBLIC KEY-----) or certificate")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		return cert.PublicKey, nil
	}
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	return nil, errors.New("couldn't parse the PEM as a public key or certificate")
}

func hashFor(alg string) (crypto.Hash, bool) {
	switch {
	case strings.HasSuffix(alg, "256"):
		return crypto.SHA256, true
	case strings.HasSuffix(alg, "384"):
		return crypto.SHA384, true
	case strings.HasSuffix(alg, "512"):
		return crypto.SHA512, true
	}
	return 0, false
}

func digest(h crypto.Hash, b []byte) []byte {
	switch h {
	case crypto.SHA256:
		s := sha256.Sum256(b)
		return s[:]
	case crypto.SHA384:
		s := sha512.Sum384(b)
		return s[:]
	case crypto.SHA512:
		s := sha512.Sum512(b)
		return s[:]
	}
	return nil
}

func b64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
}

func prettyJSON(b []byte) (string, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return "", err
	}
	return out.String(), nil
}

func claimString(c map[string]any, k string) string {
	if v, ok := c[k].(string); ok {
		return v
	}
	return ""
}

func claimStrings(c map[string]any, k string) []string {
	switch v := c[k].(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func claimTime(c map[string]any, k string) *time.Time {
	if f, ok := c[k].(float64); ok {
		t := time.Unix(int64(f), 0).UTC()
		return &t
	}
	return nil
}
