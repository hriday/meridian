package app

import "github.com/hriday/meridian/internal/ciscopw"

// AnalyzeCiscoPasswords scans a pasted Cisco config for password/secret
// commands, decoding reversible type-7 (and cleartext type-0) strings and
// classifying the one-way hashes by strength. Pure and offline.
func (a *App) AnalyzeCiscoPasswords(configText string) []ciscopw.Secret {
	return ciscopw.Analyze(configText)
}
