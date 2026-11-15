package lanrecon

import (
	"fmt"
	"net"
	"strings"

	"github.com/hriday/meridian/internal/ipspace"
)

// ApplyOptions selects which findings to write back. Importing new hosts is the
// safe default; changing or deprecating existing records is opt-in.
type ApplyOptions struct {
	ImportNew        bool // add "new" hosts as active /32 records
	UpdateMAC        bool // rewrite the mac tag on "mac-change" records
	DeprecateMissing bool // mark "missing" records deprecated
}

// ApplyResult counts what Apply wrote.
type ApplyResult struct {
	Imported   int `json:"imported"`
	Updated    int `json:"updated"`
	Deprecated int `json:"deprecated"`
}

// Apply writes the chosen reconciliation findings back to the store.
func Apply(store *ipspace.Store, findings []Finding, opts ApplyOptions) (ApplyResult, error) {
	var res ApplyResult
	byID := map[string]ipspace.Record{}
	for _, r := range store.Records() {
		byID[r.ID] = r
	}

	for _, f := range findings {
		switch f.Category {
		case New:
			if !opts.ImportNew {
				continue
			}
			if _, err := store.Add(hostPrefix(f.IP), "", ipspace.Active, setMACVendor(nil, f.MAC, f.Vendor), ""); err != nil {
				return res, fmt.Errorf("import %s: %w", f.IP, err)
			}
			res.Imported++
		case MACChange:
			if !opts.UpdateMAC || f.RecordID == "" {
				continue
			}
			r, ok := byID[f.RecordID]
			if !ok {
				continue
			}
			if err := store.Edit(r.ID, r.Name, r.Status, setMACVendor(r.Tags, f.MAC, vendorFromTags(r.Tags)), r.Notes); err != nil {
				return res, fmt.Errorf("update %s: %w", f.IP, err)
			}
			res.Updated++
		case Missing:
			if !opts.DeprecateMissing || f.RecordID == "" {
				continue
			}
			r, ok := byID[f.RecordID]
			if !ok {
				continue
			}
			if err := store.Edit(r.ID, r.Name, ipspace.Deprecated, r.Tags, r.Notes); err != nil {
				return res, fmt.Errorf("deprecate %s: %w", f.IP, err)
			}
			res.Deprecated++
		}
	}
	return res, nil
}

// hostPrefix turns an address into its host CIDR (/32 for IPv4, /128 for IPv6).
func hostPrefix(ip string) string {
	if p := net.ParseIP(ip); p != nil && p.To4() == nil {
		return ip + "/128"
	}
	return ip + "/32"
}

// setMACVendor returns tags with any existing mac:/vendor: entries removed and
// the given values (if non-empty) appended, leaving other tags untouched.
func setMACVendor(tags []string, mac, vendor string) []string {
	out := make([]string, 0, len(tags)+2)
	for _, t := range tags {
		lt := strings.ToLower(strings.TrimSpace(t))
		if strings.HasPrefix(lt, "mac:") || strings.HasPrefix(lt, "vendor:") {
			continue
		}
		out = append(out, t)
	}
	if mac != "" {
		out = append(out, "mac:"+mac)
	}
	if vendor != "" {
		out = append(out, "vendor:"+vendor)
	}
	return out
}

func vendorFromTags(tags []string) string {
	for _, t := range tags {
		if v, ok := strings.CutPrefix(strings.TrimSpace(t), "vendor:"); ok {
			return v
		}
	}
	return ""
}
