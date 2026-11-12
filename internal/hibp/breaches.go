package hibp

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
)

// Breach is one breached site in the Have I Been Pwned catalogue. These come from
// the free, keyless /breaches endpoint — the catalogue of *which sites* were
// breached and *what data* was exposed. (Searching whether a specific email
// address appears in them is a separate, paid, key-only API.)
type Breach struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Domain      string   `json:"domain"`
	BreachDate  string   `json:"breachDate"`
	PwnCount    int      `json:"pwnCount"`
	Description string   `json:"description"` // HTML stripped to plain text
	DataClasses []string `json:"dataClasses"`
	Verified    bool     `json:"verified"`
	Sensitive   bool     `json:"sensitive"`
	Malware     bool     `json:"malware"`
}

// the API's own (PascalCase) shape, for decoding.
type apiBreach struct {
	Name        string
	Title       string
	Domain      string
	BreachDate  string
	PwnCount    int
	Description string
	DataClasses []string
	IsVerified  bool
	IsSensitive bool
	IsMalware   bool
}

func (b apiBreach) toBreach() Breach {
	return Breach{
		Name: b.Name, Title: b.Title, Domain: b.Domain, BreachDate: b.BreachDate,
		PwnCount: b.PwnCount, Description: stripHTML(b.Description), DataClasses: b.DataClasses,
		Verified: b.IsVerified, Sensitive: b.IsSensitive, Malware: b.IsMalware,
	}
}

// Breaches returns the full breach catalogue (free, keyless). The caller can
// filter it by name/domain locally.
func Breaches(ctx context.Context, client Doer) ([]Breach, error) {
	var api []apiBreach
	if err := getJSON(ctx, client, "https://haveibeenpwned.com/api/v3/breaches", &api); err != nil {
		return nil, err
	}
	out := make([]Breach, len(api))
	for i, b := range api {
		out[i] = b.toBreach()
	}
	return out, nil
}

// LatestBreach returns the most recently added breach (free, keyless).
func LatestBreach(ctx context.Context, client Doer) (Breach, error) {
	var b apiBreach
	if err := getJSON(ctx, client, "https://haveibeenpwned.com/api/v3/latestbreach", &b); err != nil {
		return Breach{}, err
	}
	return b.toBreach(), nil
}

func getJSON(ctx context.Context, client Doer, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent) // required by HIBP; a missing UA is a 403
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't reach Have I Been Pwned")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Have I Been Pwned returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	s = htmlTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}
