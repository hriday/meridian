package tfscan

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parsePlanJSON parses `terraform show -json` output into rawResources, walking
// planned_values.root_module (and its child_modules recursively); it falls back
// to values.root_module for a state show. Each resource's resolved `values`
// object becomes its mapBody.
func parsePlanJSON(text string) ([]rawResource, error) {
	var root struct {
		PlannedValues struct {
			RootModule json.RawMessage `json:"root_module"`
		} `json:"planned_values"`
		Values struct {
			RootModule json.RawMessage `json:"root_module"`
		} `json:"values"`
	}
	if err := json.Unmarshal([]byte(text), &root); err != nil {
		return nil, fmt.Errorf("not a valid JSON document: %w", err)
	}
	rm := root.PlannedValues.RootModule
	if len(rm) == 0 {
		rm = root.Values.RootModule
	}
	if len(rm) == 0 {
		return nil, fmt.Errorf("this JSON doesn't look like a `terraform show -json` plan (no planned_values/values root_module)")
	}
	var out []rawResource
	if err := walkModule(rm, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func walkModule(raw json.RawMessage, out *[]rawResource) error {
	var mod struct {
		Resources []struct {
			Type         string         `json:"type"`
			Name         string         `json:"name"`
			ProviderName string         `json:"provider_name"`
			Values       map[string]any `json:"values"`
		} `json:"resources"`
		ChildModules []json.RawMessage `json:"child_modules"`
	}
	if err := json.Unmarshal(raw, &mod); err != nil {
		return fmt.Errorf("terraform plan module: %w", err)
	}
	for _, r := range mod.Resources {
		*out = append(*out, rawResource{
			typ:      r.Type,
			name:     r.Name,
			provider: shortProvider(r.ProviderName),
			body:     mapBody(r.Values),
		})
	}
	for _, cm := range mod.ChildModules {
		if err := walkModule(cm, out); err != nil {
			return err
		}
	}
	return nil
}

// shortProvider reduces "registry.terraform.io/hashicorp/aws" to "aws".
func shortProvider(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
