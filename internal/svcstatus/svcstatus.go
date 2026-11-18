// Package svcstatus answers "is this online service having an outage right now?"
// from a service's own authoritative status feed — free and keyless, the real
// answer to a Downdetector-style question (without the crowd reports).
//
// Most services publish an Atlassian Statuspage feed (/api/v2/summary.json), and
// that's the default. A few big providers run their own status pages with their
// own feeds; those get a per-provider adapter (see apple.go, google.go,
// azure.go) that maps the provider's feed onto the same Status model.
package svcstatus

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Doer is the slice of *http.Client used, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

const userAgent = "meridian (+https://github.com/hriday/meridian)"

// provider describes where and how to read one service's status.
type provider struct {
	Name string // display name, e.g. "Apple"
	Kind string // statuspage | apple | google | azure
	URL  string // statuspage base, or the custom provider's feed URL
}

// known maps a short slug (also the GUI chip label) to its provider.
var known = map[string]provider{
	// Atlassian Statuspage services (the default kind).
	"github":       {"GitHub", kindStatuspage, "https://www.githubstatus.com"},
	"cloudflare":   {"Cloudflare", kindStatuspage, "https://www.cloudflarestatus.com"},
	"discord":      {"Discord", kindStatuspage, "https://discordstatus.com"},
	"reddit":       {"Reddit", kindStatuspage, "https://www.redditstatus.com"},
	"dropbox":      {"Dropbox", kindStatuspage, "https://status.dropbox.com"},
	"twilio":       {"Twilio", kindStatuspage, "https://status.twilio.com"},
	"digitalocean": {"DigitalOcean", kindStatuspage, "https://status.digitalocean.com"},
	"datadog":      {"Datadog", kindStatuspage, "https://status.datadoghq.com"},
	"atlassian":    {"Atlassian", kindStatuspage, "https://status.atlassian.com"},
	"npm":          {"npm", kindStatuspage, "https://status.npmjs.org"},
	"coinbase":     {"Coinbase", kindStatuspage, "https://status.coinbase.com"},
	"openai":       {"OpenAI", kindStatuspage, "https://status.openai.com"},
	"anthropic":    {"Anthropic", kindStatuspage, "https://status.claude.com"},
	"zoom":         {"Zoom", kindStatuspage, "https://www.zoomstatus.com"},
	"vercel":       {"Vercel", kindStatuspage, "https://www.vercel-status.com"},
	"netlify":      {"Netlify", kindStatuspage, "https://www.netlifystatus.com"},
	"figma":        {"Figma", kindStatuspage, "https://status.figma.com"},
	"docker":       {"Docker", kindStatuspage, "https://www.dockerstatus.com"},
	"mongodb":      {"MongoDB", kindStatuspage, "https://status.mongodb.com"},
	"hashicorp":    {"HashiCorp", kindStatuspage, "https://status.hashicorp.com"},
	"pypi":         {"PyPI", kindStatuspage, "https://status.python.org"},
	"squarespace":  {"Squarespace", kindStatuspage, "https://status.squarespace.com"},

	// Custom status pages with their own feeds (per-provider adapters).
	"apple":            {"Apple", kindApple, appleConsumerFeed},
	"apple-developer":  {"Apple Developer", kindApple, appleDeveloperFeed},
	"azure":            {"Microsoft Azure", kindAzure, azureFeed},
	"google-cloud":     {"Google Cloud", kindGoogle, "https://status.cloud.google.com/incidents.json"},
	"google-workspace": {"Google Workspace", kindGoogle, "https://www.google.com/appsstatus/dashboard/incidents.json"},
}

const (
	kindStatuspage = "statuspage"
	kindApple      = "apple"
	kindGoogle     = "google"
	kindAzure      = "azure"
)

// KnownSlugs returns the known service slugs, sorted (also the GUI chip list).
func KnownSlugs() []string {
	out := make([]string, 0, len(known))
	for k := range known {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type Component struct {
	Name   string `json:"name"`
	Status string `json:"status"` // operational | degraded_performance | partial_outage | major_outage | under_maintenance
}

type Incident struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // investigating | identified | monitoring | ongoing
	Impact    string `json:"impact"` // none | minor | major | critical
	Shortlink string `json:"shortlink"`
	UpdatedAt string `json:"updatedAt"`
}

// Status is the distilled result for one service, shared across every provider.
type Status struct {
	Service     string      `json:"service"`
	URL         string      `json:"url"`
	Source      string      `json:"source"`      // which feed answered: Statuspage | Apple | Google | Azure
	Indicator   string      `json:"indicator"`   // none | minor | major | critical
	Description string      `json:"description"` // e.g. "All Systems Operational"
	Operational bool        `json:"operational"` // indicator == none
	Impacted    []Component `json:"impacted"`    // components not fully operational
	Incidents   []Incident  `json:"incidents"`   // unresolved incidents
}

// Check fetches and distills a service's status. target is a known slug, a
// status-page host, or a full URL; the provider is detected and the matching
// adapter handles the feed.
func Check(ctx context.Context, client Doer, target string) (Status, error) {
	p, err := resolveProvider(target)
	if err != nil {
		return Status{}, err
	}
	switch p.Kind {
	case kindApple:
		return checkApple(ctx, client, p)
	case kindGoogle:
		return checkGoogle(ctx, client, p)
	case kindAzure:
		return checkAzure(ctx, client, p)
	default:
		return checkStatuspage(ctx, client, p)
	}
}

// resolveProvider turns a slug / host / URL into a provider. Known slugs win;
// otherwise the host is matched against the custom providers, and anything else
// is assumed to be a Statuspage-backed page.
func resolveProvider(target string) (provider, error) {
	t := strings.ToLower(strings.TrimSpace(target))
	if t == "" {
		return provider{}, fmt.Errorf("enter a service name or status-page URL")
	}
	if p, ok := known[t]; ok {
		return p, nil
	}

	host := t
	full := t
	if strings.Contains(t, "://") {
		u, err := url.Parse(t)
		if err != nil || u.Host == "" {
			return provider{}, fmt.Errorf("that doesn't look like a URL")
		}
		host = strings.ToLower(u.Host)
	} else if !strings.Contains(t, ".") {
		return provider{}, fmt.Errorf("unknown service %q — paste its status-page URL, or try one of: %s",
			target, strings.Join(KnownSlugs(), ", "))
	}

	switch {
	case strings.Contains(host, "apple.com"):
		if strings.Contains(host, "developer") || strings.Contains(full, "developer") {
			return known["apple-developer"], nil
		}
		return known["apple"], nil
	case strings.Contains(host, "azure") || strings.Contains(host, "azure.status.microsoft"):
		return known["azure"], nil
	case strings.Contains(host, "status.cloud.google.com"):
		return known["google-cloud"], nil
	case strings.Contains(host, "google.com") && strings.Contains(full, "appsstatus"):
		return known["google-workspace"], nil
	case isMicrosoft365(host):
		return provider{}, fmt.Errorf("Microsoft 365 service health needs an admin sign-in — Microsoft publishes no public, unauthenticated feed. Check it by hand at https://status.cloud.microsoft")
	case strings.Contains(host, "aws.amazon") || strings.Contains(host, "status.aws") || strings.Contains(host, "health.aws"):
		return provider{}, fmt.Errorf("AWS only publishes an RSS feed with no machine-readable \"resolved\" flag, so Meridian can't give a reliable up/down yet. Check it by hand at https://health.aws.amazon.com/health/status")
	default:
		// Assume Statuspage: use scheme://host as the base.
		base := "https://" + host
		if strings.Contains(t, "://") {
			if u, err := url.Parse(t); err == nil {
				base = u.Scheme + "://" + u.Host
			}
		}
		return provider{Name: host, Kind: kindStatuspage, URL: strings.TrimSuffix(base, "/")}, nil
	}
}

func isMicrosoft365(host string) bool {
	for _, h := range []string{"office365.com", "status.office.com", "portal.office.com", "status.cloud.microsoft", "office.com"} {
		if strings.Contains(host, h) {
			return true
		}
	}
	return false
}
