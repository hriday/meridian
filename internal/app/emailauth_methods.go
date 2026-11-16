package app

import (
	"context"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/cli"
	"github.com/hriday/meridian/internal/emailauth"
)

// EmailAuthCheck resolves and grades a domain's outbound-mail authentication —
// SPF, DKIM (across common or caller-supplied selectors), and DMARC — from DNS.
// selectorsCSV is an optional comma-separated selector list; empty uses the
// built-in default set.
func (a *App) EmailAuthCheck(domain, selectorsCSV string) emailauth.Report {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var selectors []string
	for _, s := range strings.Split(selectorsCSV, ",") {
		if s = strings.TrimSpace(s); s != "" {
			selectors = append(selectors, s)
		}
	}
	// Use the robust dnsx-based TXT lookup (the Go stdlib resolver fails on some
	// large TXT responses, which made emailauth report a present SPF as absent).
	return emailauth.Check(ctx, cli.LookupTXT, domain, selectors)
}
