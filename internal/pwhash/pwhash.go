// Package pwhash generates and verifies bcrypt password hashes locally — so a
// password never has to be pasted into a web bcrypt/htpasswd generator (where it
// would be sent to someone else's server). Handy for nginx/Apache basic-auth
// (htpasswd) entries. Point-in-time; nothing is stored.
package pwhash

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// MinCost / MaxCost / DefaultCost mirror bcrypt's bounds.
const (
	MinCost     = bcrypt.MinCost
	MaxCost     = bcrypt.MaxCost
	DefaultCost = bcrypt.DefaultCost
)

// GenResult is a generated hash, optionally formatted as an htpasswd line.
type GenResult struct {
	Hash     string `json:"hash"`
	Cost     int    `json:"cost"`
	Htpasswd string `json:"htpasswd,omitempty"` // "user:hash" when a user is given
}

// Generate makes a bcrypt hash of password at the given cost (0 => DefaultCost).
// If user is non-empty the result also carries an htpasswd line.
func Generate(password, user string, cost int) (GenResult, error) {
	if password == "" {
		return GenResult{}, fmt.Errorf("enter a password to hash")
	}
	if cost == 0 {
		cost = DefaultCost
	}
	if cost < MinCost || cost > MaxCost {
		return GenResult{}, fmt.Errorf("cost must be between %d and %d", MinCost, MaxCost)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return GenResult{}, err
	}
	res := GenResult{Hash: string(h), Cost: cost}
	if user != "" {
		res.Htpasswd = user + ":" + string(h)
	}
	return res, nil
}

// CheckResult is the outcome of verifying a password against a hash.
type CheckResult struct {
	Match bool `json:"match"`
	Cost  int  `json:"cost,omitempty"`
}

// Check reports whether password matches a bcrypt hash. An "htpasswd" line
// (user:hash) is accepted too — the hash part is used.
func Check(password, hash string) (CheckResult, error) {
	hash = strings.TrimSpace(hash)
	if i := strings.Index(hash, ":$2"); i >= 0 { // user:$2...  htpasswd line
		hash = hash[i+1:]
	}
	if !strings.HasPrefix(hash, "$2") {
		return CheckResult{}, fmt.Errorf("not a bcrypt hash (expected it to start with $2a$/$2b$/$2y$)")
	}
	cost, _ := bcrypt.Cost([]byte(hash))
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return CheckResult{Match: true, Cost: cost}, nil
	}
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return CheckResult{Match: false, Cost: cost}, nil
	}
	return CheckResult{}, err
}
