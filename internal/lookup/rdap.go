package lookup

import (
	"context"
	"encoding/json"
	"strings"
)

// rdapBase is the RDAP bootstrap/redirector root (rdap.org forwards to the
// authoritative RIR). It is a package var so tests can override it.
var rdapBase = "https://rdap.org/"

// RDAPEntity is the slice of RDAP we surface: who a resource is registered to.
type RDAPEntity struct {
	Handle     string `json:"handle"`
	Org        string `json:"org"`
	AbuseEmail string `json:"abuseEmail"`
	Registry   string `json:"registry"`
}

// rdapRaw and rdapEntityRaw decode the parts of an RDAP object we read. RDAP
// bodies vary across RIRs, so every field is optional and parsing never
// assumes a shape — missing pieces yield empty strings, not errors.
type rdapRaw struct {
	Handle   string          `json:"handle"`
	Name     string          `json:"name"`
	Port43   string          `json:"port43"`
	Entities []rdapEntityRaw `json:"entities"`
}

type rdapEntityRaw struct {
	Handle     string            `json:"handle"`
	Roles      []string          `json:"roles"`
	VcardArray []json.RawMessage `json:"vcardArray"`
	Entities   []rdapEntityRaw   `json:"entities"`
}

func (c *Client) rdapIP(ctx context.Context, ip string) (RDAPEntity, error) {
	return c.rdap(ctx, rdapBase+"ip/"+ip)
}

func (c *Client) rdapASN(ctx context.Context, asn string) (RDAPEntity, error) {
	num := strings.TrimLeft(strings.ToUpper(strings.TrimSpace(asn)), "AS")
	return c.rdap(ctx, rdapBase+"autnum/"+num)
}

func (c *Client) rdap(ctx context.Context, url string) (RDAPEntity, error) {
	var raw rdapRaw
	if err := c.GetJSON(ctx, url, &raw); err != nil {
		return RDAPEntity{}, err
	}
	e := RDAPEntity{Handle: raw.Handle, Registry: registryFromPort43(raw.Port43)}
	if org := findRole(raw.Entities, "registrant", "fn"); org != "" {
		e.Org = org
	} else {
		e.Org = raw.Name // best effort when the registrant carries no vCard
	}
	e.AbuseEmail = findRole(raw.Entities, "abuse", "email")
	return e, nil
}

// findRole walks the entity tree (entities nest) for the first entity holding
// role, and returns the named vCard field (e.g. "fn", "email").
func findRole(entities []rdapEntityRaw, role, field string) string {
	for _, ent := range entities {
		for _, r := range ent.Roles {
			if strings.EqualFold(r, role) {
				if v := vcardField(ent.VcardArray, field); v != "" {
					return v
				}
			}
		}
		if v := findRole(ent.Entities, role, field); v != "" {
			return v
		}
	}
	return ""
}

// vcardField pulls a field value out of a jCard ["vcard",[[name,params,type,value],…]].
// It guards every index and type so a malformed vCard yields "" rather than a panic.
func vcardField(vcardArray []json.RawMessage, field string) string {
	if len(vcardArray) < 2 {
		return ""
	}
	var rows [][]json.RawMessage
	if err := json.Unmarshal(vcardArray[1], &rows); err != nil {
		return ""
	}
	for _, row := range rows {
		if len(row) < 4 {
			continue
		}
		var name string
		if json.Unmarshal(row[0], &name) != nil || !strings.EqualFold(name, field) {
			continue
		}
		var val string
		if json.Unmarshal(row[3], &val) == nil {
			return val
		}
	}
	return ""
}

func registryFromPort43(p string) string {
	switch {
	case strings.Contains(p, "arin.net"):
		return "ARIN"
	case strings.Contains(p, "ripe.net"):
		return "RIPE NCC"
	case strings.Contains(p, "apnic.net"):
		return "APNIC"
	case strings.Contains(p, "lacnic.net"):
		return "LACNIC"
	case strings.Contains(p, "afrinic.net"):
		return "AFRINIC"
	default:
		return ""
	}
}
