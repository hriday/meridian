// Package hibp checks NTLM password hashes against the Have I Been Pwned
// "Pwned Passwords" service using k-anonymity: only the first 5 hex characters
// of each hash are ever sent. The service returns every hash suffix sharing that
// prefix (padded with decoys), and the match is made locally — the password and
// its full hash never leave the machine. The API is free and needs no key.
package hibp

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	rangeURL  = "https://api.pwnedpasswords.com/range/"
	userAgent = "meridian (+https://github.com/hriday/meridian)"
)

// Doer is the slice of *http.Client the checker uses, so tests can inject a fake.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// CheckNTLM checks NTLM (NT) hashes against Pwned Passwords in NTLM mode and
// returns a map from each input hash to the number of times it appears in known
// breaches (0 = not found). Each distinct 5-character prefix is queried once, and
// only that prefix leaves the machine; the 27-character suffix is matched
// locally. Input hashes may be any case; the returned map is keyed by the exact
// input strings.
func CheckNTLM(ctx context.Context, client Doer, hashes []string) (map[string]int, error) {
	out := make(map[string]int, len(hashes))
	byPrefix := map[string][]string{} // prefix(upper) -> input hashes with that prefix
	for _, h := range hashes {
		u := strings.ToUpper(strings.TrimSpace(h))
		if len(u) != 32 || !isHex(u) {
			continue // not an NTLM hash
		}
		byPrefix[u[:5]] = append(byPrefix[u[:5]], h)
	}
	for prefix, inputs := range byPrefix {
		counts, err := rangeNTLM(ctx, client, prefix)
		if err != nil {
			return out, err
		}
		for _, h := range inputs {
			suffix := strings.ToUpper(strings.TrimSpace(h))[5:]
			out[h] = counts[suffix]
		}
	}
	return out, nil
}

// rangeNTLM fetches all NTLM-mode suffixes for one prefix and returns suffix→count.
func rangeNTLM(ctx context.Context, client Doer, prefix string) (map[string]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rangeURL+prefix+"?mode=ntlm", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	// Pad the response with decoys so a passive observer can't infer the prefix
	// from the response size.
	req.Header.Set("Add-Padding", "true")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't reach Pwned Passwords")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Pwned Passwords returned %s", resp.Status)
	}

	counts := map[string]int{}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		i := strings.IndexByte(line, ':')
		if i < 0 {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(line[i+1:]))
		if err != nil || n == 0 {
			continue // padding decoys carry a count of 0
		}
		counts[strings.ToUpper(line[:i])] = n
	}
	return counts, sc.Err()
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
