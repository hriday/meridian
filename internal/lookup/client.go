package lookup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client fetches JSON over an injected Doer, caching successful responses by
// URL for a TTL so repeated lookups of the same entity don't re-hit the net.
type Client struct {
	doer Doer
	ttl  time.Duration

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	body    []byte
	expires time.Time
}

// NewClient returns a Client over d, caching responses for ttl.
func NewClient(d Doer, ttl time.Duration) *Client {
	return &Client{doer: d, ttl: ttl, cache: map[string]cacheEntry{}}
}

// GetJSON fetches url and decodes the JSON body into v. Successful bodies are
// cached for the client's TTL; a cache hit decodes without any network call.
// A non-2xx status or an undecodable body is returned as an error.
func (c *Client) GetJSON(ctx context.Context, url string, v any) error {
	if body, ok := c.cached(url); ok {
		return json.Unmarshal(body, v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8 MiB cap
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("GET %s: decode: %w", url, err)
	}
	c.store(url, body)
	return nil
}

func (c *Client) cached(url string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.cache[url]
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.body, true
}

func (c *Client) store(url string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[url] = cacheEntry{body: body, expires: time.Now().Add(c.ttl)}
}
