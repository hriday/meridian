package lookup

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ripePrefixOverview returns the origin ASN(s), the holder, and whether the
// prefix is announced, from RIPEstat's prefix-overview.
func (c *Client) ripePrefixOverview(ctx context.Context, prefix string) (origins []string, holder string, announced bool, err error) {
	var r struct {
		Data struct {
			Announced bool `json:"announced"`
			ASNs      []struct {
				ASN    int    `json:"asn"`
				Holder string `json:"holder"`
			} `json:"asns"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, ripeURL("prefix-overview", prefix), &r); err != nil {
		return nil, "", false, err
	}
	for i, a := range r.Data.ASNs {
		origins = append(origins, strconv.Itoa(a.ASN))
		if i == 0 {
			holder = a.Holder
		}
	}
	return origins, holder, r.Data.Announced, nil
}

// ripeRelatedPrefixes returns the less- and more-specific prefixes around one.
func (c *Client) ripeRelatedPrefixes(ctx context.Context, prefix string) (less, more []string, err error) {
	var r struct {
		Data struct {
			Prefixes []struct {
				Prefix       string `json:"prefix"`
				Relationship string `json:"relationship"`
			} `json:"prefixes"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, ripeURL("related-prefixes", prefix), &r); err != nil {
		return nil, nil, err
	}
	for _, p := range r.Data.Prefixes {
		// RIPEstat uses "Overlap - Less Specific" / "Overlap - More Specific";
		// "Adjacency - Left/Right" are neighbouring (not nesting) blocks — dropped.
		rel := strings.ToLower(p.Relationship)
		switch {
		case strings.Contains(rel, "less specific"):
			less = append(less, p.Prefix)
		case strings.Contains(rel, "more specific"):
			more = append(more, p.Prefix)
		}
	}
	return less, more, nil
}

// ripeRPKI returns the RPKI validation state (valid/invalid/unknown) for a
// prefix+origin and a short description of the matching ROA, via RIPEstat.
func (c *Client) ripeRPKI(ctx context.Context, prefix, asn string) (state, roa string, err error) {
	num := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(asn)), "AS")
	u := fmt.Sprintf("%srpki-validation/data.json?resource=%s&prefix=%s", ripeBase, url.QueryEscape(num), url.QueryEscape(prefix))
	var r struct {
		Data struct {
			Status        string `json:"status"`
			ValidatingRoas []struct {
				Prefix    string `json:"prefix"`
				MaxLength int    `json:"max_length"`
				Origin    string `json:"origin"`
			} `json:"validating_roas"`
		} `json:"data"`
	}
	if err := c.GetJSON(ctx, u, &r); err != nil {
		return "", "", err
	}
	if len(r.Data.ValidatingRoas) > 0 {
		v := r.Data.ValidatingRoas[0]
		roa = fmt.Sprintf("%s max /%d (AS%s)", v.Prefix, v.MaxLength, v.Origin)
	}
	return r.Data.Status, roa, nil
}
