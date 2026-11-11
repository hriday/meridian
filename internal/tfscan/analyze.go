package tfscan

import (
	"fmt"
	"strings"
	"unicode"
)

// Analyze reads a Terraform config offline and returns the exposure analysis.
// source is "hcl", "json", or "" to auto-detect (a leading '{' means a
// `terraform show -json` plan; otherwise HCL).
func Analyze(text, source string) (Analysis, error) {
	if strings.TrimSpace(text) == "" {
		return Analysis{}, fmt.Errorf("no Terraform config provided — paste .tf files or `terraform show -json` output")
	}
	src := strings.ToLower(strings.TrimSpace(source))
	if src == "" {
		if firstNonSpace(text) == '{' {
			src = "json"
		} else {
			src = "hcl"
		}
	}

	var (
		rs  []rawResource
		err error
	)
	switch src {
	case "json":
		rs, err = parsePlanJSON(text)
	case "hcl":
		rs, err = parseHCL(text)
	default:
		return Analysis{}, fmt.Errorf("source %q is not supported — use hcl or json", source)
	}
	if err != nil {
		return Analysis{}, err
	}
	a := check(rs)
	a.Source = src
	if anyDynamic(a.Resources) {
		a.Notes = append(a.Notes, "Some values come from variables, locals or modules and couldn't be resolved from HCL — analyze a `terraform show -json` plan for a concrete result.")
	}
	return a, nil
}

func anyDynamic(rs []Resource) bool {
	for _, r := range rs {
		if r.Dynamic {
			return true
		}
	}
	return false
}

func firstNonSpace(s string) rune {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return r
		}
	}
	return 0
}
