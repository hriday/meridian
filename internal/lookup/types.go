// Package lookup resolves an IP address or ASN into a source-labelled report
// ("who owns this") from live public data — RDAP for registry/ownership and
// RIPEstat for routing. The engine is UI-free; the Wails controller and CLI
// are thin adapters over it. Every section of a report carries its own
// Source so the UI can show where a fact came from and how fresh it is, and
// a failed source degrades that section rather than failing the lookup.
package lookup

import (
	"net/http"
	"time"
)

// userAgent is sent on every outbound request. It never carries user identity.
const userAgent = "meridian/0.1 (+https://github.com/hriday/meridian)"

// Doer is the minimal HTTP surface the engine needs; tests inject a fake.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Source labels where a section of a report came from and when it was fetched.
// Err is empty on success and holds a human-readable reason on failure.
type Source struct {
	Label     string    `json:"label"`
	FetchedAt time.Time `json:"fetchedAt"`
	Err       string    `json:"err"`
}
