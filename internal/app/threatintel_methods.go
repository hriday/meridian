package app

import (
	"context"
	"net/http"
	"time"

	"github.com/hriday/meridian/internal/threatintel"
)

// ThreatIntelCheck takes a free-form blob of IP addresses (pasted, one per line
// or comma/space separated), pulls out the valid ones, and checks each against
// abuse.ch's Feodo Tracker botnet C2 blocklist. Keyless — nothing but the IPs
// you paste is involved, and the feed is a public download.
func (a *App) ThreatIntelCheck(input string) ([]threatintel.Hit, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ips := threatintel.ExtractIPs(input)
	return threatintel.Check(ctx, &http.Client{Timeout: 25 * time.Second}, ips)
}
