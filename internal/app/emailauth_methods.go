package app

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/emailauth"
)

// EmailAuthCheck resolves and grades a domain's outbound-mail authentication —
// SPF, DKIM (across common or caller-supplied selectors), and DMARC — from DNS.
// selectorsCSV is an optional comma-separated selector list; empty uses the
// built-in default set.
func (a *App) EmailAuthCheck(domain, selectorsCSV string) emailauth.Report {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	lookup := func(ctx context.Context, name string) ([]string, error) {
		return net.DefaultResolver.LookupTXT(ctx, name)
	}

	var selectors []string
	for _, s := range strings.Split(selectorsCSV, ",") {
		if s = strings.TrimSpace(s); s != "" {
			selectors = append(selectors, s)
		}
	}
	return emailauth.Check(ctx, lookup, domain, selectors)
}
