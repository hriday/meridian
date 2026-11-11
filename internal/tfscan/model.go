// Package tfscan reads Terraform configs (HCL .tf or `terraform show -json`)
// offline and surfaces what the infrastructure exposes to the internet — open
// security-group rules, public buckets and databases, unencrypted storage —
// plus a findings list and an exposure graph. It does no I/O and never connects
// to a cloud account or reads state.
package tfscan

import (
	"encoding/json"
	"strconv"
)

// rawBody is the parser-agnostic accessor over one resource's (or nested
// block's) attributes. Both the HCL and plan-JSON parsers implement it, so the
// checks engine is written once. A list coerces a scalar to a one-element slice;
// the bool reports whether the value was fully resolved (an HCL value drawn from
// a variable/function is unresolved).
type rawBody interface {
	str(name string) (string, bool)
	list(name string) ([]string, bool)
	blocks(name string) []rawBody
}

// rawResource is one Terraform resource from either parser.
type rawResource struct {
	typ, name, provider, file string
	line                      int
	dynamic                   bool
	body                      rawBody
}

// mapBody is a rawBody over a decoded JSON object (the plan parser uses it, and
// tests build resources with it directly).
type mapBody map[string]any

func scalarStr(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		if t {
			return "true", true
		}
		return "false", true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case int:
		return strconv.Itoa(t), true
	case json.Number:
		return t.String(), true
	}
	return "", false
}

func (m mapBody) str(name string) (string, bool) {
	v, ok := m[name]
	if !ok {
		return "", false
	}
	return scalarStr(v)
}

func (m mapBody) list(name string) ([]string, bool) {
	v, ok := m[name]
	if !ok {
		return nil, false
	}
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := scalarStr(e); ok {
				out = append(out, s)
			}
		}
		return out, true
	default:
		if s, ok := scalarStr(v); ok {
			return []string{s}, true
		}
	}
	return nil, false
}

func (m mapBody) blocks(name string) []rawBody {
	v, ok := m[name]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []any:
		var out []rawBody
		for _, e := range t {
			if mm, ok := e.(map[string]any); ok {
				out = append(out, mapBody(mm))
			}
		}
		return out
	case map[string]any:
		return []rawBody{mapBody(t)}
	}
	return nil
}

// --- public analysis model ---

// Resource is one parsed resource for the table/drill-in.
type Resource struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Provider string   `json:"provider"`
	Public   bool     `json:"public"`
	Exposure []string `json:"exposure"` // human labels, never a secret
	Dynamic  bool     `json:"dynamic"`  // a security value was unresolved (HCL var)
	File     string   `json:"file"`
	Line     int      `json:"line"`
}

// Finding is one exposure or misconfiguration.
type Finding struct {
	Severity string `json:"severity"` // critical | warning | info
	Category string `json:"category"` // exposure | encryption | best-practice
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Resource string `json:"resource"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// ExposureEdge is one Internet→resource edge for the diagram.
type ExposureEdge struct {
	Target   string `json:"target"`
	Label    string `json:"label"`
	Severity string `json:"severity"`
}

// Counts are the headline numbers.
type Counts struct {
	Resources      int `json:"resources"`
	SecurityGroups int `json:"securityGroups"`
	Exposed        int `json:"exposed"`
	Critical       int `json:"critical"`
	Findings       int `json:"findings"`
}

// Analysis is the result for one config.
type Analysis struct {
	Source    string         `json:"source"` // hcl | json
	Resources []Resource     `json:"resources"`
	Findings  []Finding      `json:"findings"`
	Exposure  []ExposureEdge `json:"exposure"`
	Counts    Counts         `json:"counts"`
	Notes     []string       `json:"notes"`
}

func newAnalysis(source string) Analysis {
	return Analysis{
		Source:    source,
		Resources: []Resource{},
		Findings:  []Finding{},
		Exposure:  []ExposureEdge{},
		Notes:     []string{},
	}
}
