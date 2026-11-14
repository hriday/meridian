// Package totp generates TOTP (RFC 6238) two-factor codes locally from a shared
// secret — so a 2FA seed never has to be pasted into a web TOTP generator (where
// it would be sent to someone else's server, defeating the point of 2FA). Handy
// for testing a 2FA setup. Point-in-time; nothing is stored.
package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
)

// Config is a TOTP parameter set.
type Config struct {
	Secret    string `json:"-"` // base32, never serialized back out
	Digits    int    `json:"digits"`
	Period    int    `json:"period"` // seconds
	Algorithm string `json:"algorithm"`
	Issuer    string `json:"issuer,omitempty"`
	Account   string `json:"account,omitempty"`
}

// Code is a generated TOTP value at a moment in time.
type Code struct {
	Code        string `json:"code"`
	SecondsLeft int    `json:"secondsLeft"` // until this code rolls over
	Period      int    `json:"period"`
	Issuer      string `json:"issuer,omitempty"`
	Account     string `json:"account,omitempty"`
}

// Parse accepts either a bare base32 secret or a full otpauth:// URI (as encoded
// in a 2FA QR code) and fills in the defaults (6 digits, 30s, SHA1).
func Parse(input string) (Config, error) {
	input = strings.TrimSpace(input)
	cfg := Config{Digits: 6, Period: 30, Algorithm: "SHA1"}
	if strings.HasPrefix(strings.ToLower(input), "otpauth://") {
		u, err := url.Parse(input)
		if err != nil {
			return Config{}, fmt.Errorf("bad otpauth URI: %w", err)
		}
		q := u.Query()
		cfg.Secret = q.Get("secret")
		if d := q.Get("digits"); d != "" {
			if n, err := strconv.Atoi(d); err == nil {
				cfg.Digits = n
			}
		}
		if p := q.Get("period"); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				cfg.Period = n
			}
		}
		if a := q.Get("algorithm"); a != "" {
			cfg.Algorithm = strings.ToUpper(a)
		}
		cfg.Issuer = q.Get("issuer")
		label := strings.TrimPrefix(u.Path, "/")
		if label != "" {
			if i := strings.Index(label, ":"); i >= 0 {
				if cfg.Issuer == "" {
					cfg.Issuer = label[:i]
				}
				cfg.Account = strings.TrimSpace(label[i+1:])
			} else {
				cfg.Account = label
			}
		}
	} else {
		cfg.Secret = input
	}
	if strings.TrimSpace(cfg.Secret) == "" {
		return Config{}, fmt.Errorf("no secret found — paste a base32 secret or an otpauth:// URI")
	}
	return cfg, nil
}

// Generate returns the TOTP code for cfg at unix time now (seconds).
func Generate(cfg Config, nowUnix int64) (Code, error) {
	if cfg.Period <= 0 {
		cfg.Period = 30
	}
	if cfg.Digits <= 0 {
		cfg.Digits = 6
	}
	key, err := decodeSecret(cfg.Secret)
	if err != nil {
		return Code{}, err
	}
	h, err := hashFor(cfg.Algorithm)
	if err != nil {
		return Code{}, err
	}
	counter := uint64(nowUnix / int64(cfg.Period))
	code := hotp(key, counter, cfg.Digits, h)
	left := cfg.Period - int(nowUnix%int64(cfg.Period))
	return Code{Code: code, SecondsLeft: left, Period: cfg.Period, Issuer: cfg.Issuer, Account: cfg.Account}, nil
}

func hotp(key []byte, counter uint64, digits int, h func() hash.Hash) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(h, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := (uint32(sum[off]&0x7f) << 24) | (uint32(sum[off+1]) << 16) |
		(uint32(sum[off+2]) << 8) | uint32(sum[off+3])
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, bin%mod)
}

func decodeSecret(s string) ([]byte, error) {
	s = strings.ToUpper(strings.NewReplacer(" ", "", "-", "", "=", "").Replace(s))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("secret isn't valid base32: %w", err)
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("secret is empty")
	}
	return key, nil
}

func hashFor(alg string) (func() hash.Hash, error) {
	switch strings.ToUpper(strings.TrimSpace(alg)) {
	case "", "SHA1":
		return sha1.New, nil
	case "SHA256":
		return sha256.New, nil
	case "SHA512":
		return sha512.New, nil
	}
	return nil, fmt.Errorf("unsupported algorithm %q (want SHA1, SHA256, or SHA512)", alg)
}
