package ipspace

import (
	"fmt"
	"io"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// NetBox import/export. NetBox is the common open-source IPAM; its prefixes
// export (API YAML, pynetbox, or an Ansible inventory) is a list of objects with
// a `prefix`, a `status` (a string or a {value,label} object), a `description`,
// and `tags` (strings or {name,slug} objects). We read all of those shapes, and
// our export writes the simple list form NetBox and humans both accept.

type nbPrefix struct {
	Prefix      string      `yaml:"prefix"`
	Status      interface{} `yaml:"status,omitempty"`
	Description string      `yaml:"description,omitempty"`
	Tags        interface{} `yaml:"tags,omitempty"`
}

type nbWrapper struct {
	Results  []nbPrefix `yaml:"results"`
	Prefixes []nbPrefix `yaml:"prefixes"`
}

// ImportNetBoxYAML adds records from a NetBox prefixes export. A bad CIDR or an
// overlap conflict skips that row and is recorded, never aborting the batch.
func (s *Store) ImportNetBoxYAML(r io.Reader) (ImportReport, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return ImportReport{}, err
	}
	var list []nbPrefix
	if err := yaml.Unmarshal(data, &list); err != nil || len(list) == 0 {
		var w nbWrapper
		if err2 := yaml.Unmarshal(data, &w); err2 == nil {
			if len(w.Results) > 0 {
				list = w.Results
			} else if len(w.Prefixes) > 0 {
				list = w.Prefixes
			}
		}
		if len(list) == 0 && err != nil {
			return ImportReport{}, fmt.Errorf("not a NetBox prefixes export: %w", err)
		}
	}

	var rep ImportReport
	for _, p := range list {
		prefix := strings.TrimSpace(p.Prefix)
		if prefix == "" {
			continue
		}
		if _, err := s.Add(prefix, strings.TrimSpace(p.Description), nbStatus(p.Status), nbTags(p.Tags), ""); err != nil {
			rep.Skipped++
			rep.Errors = append(rep.Errors, fmt.Sprintf("%s: %v", prefix, err))
			continue
		}
		rep.Added++
	}
	return rep, nil
}

// ExportNetBoxYAML writes every record as a NetBox-style prefixes list.
func (s *Store) ExportNetBoxYAML(w io.Writer) error {
	recs := s.Records()
	out := make([]nbPrefix, 0, len(recs))
	for _, r := range recs {
		var tags interface{}
		if len(r.Tags) > 0 {
			tags = r.Tags
		}
		out = append(out, nbPrefix{
			Prefix:      r.Prefix,
			Status:      string(r.Status),
			Description: r.Name,
			Tags:        tags,
		})
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(out); err != nil {
		return err
	}
	return enc.Close()
}

// nbStatus maps a NetBox status (string or {value} object) to ours. NetBox's
// "container"/"available" have no direct equivalent, so they fall back sensibly.
func nbStatus(v interface{}) Status {
	s := ""
	switch t := v.(type) {
	case string:
		s = t
	case map[string]interface{}:
		if val, ok := t["value"].(string); ok {
			s = val
		}
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "active", "container":
		return Active
	case "reserved", "available":
		return Reserved
	case "deprecated":
		return Deprecated
	default:
		return Active
	}
}

// nbTags reads tags as a list of strings or of {name,slug} objects.
func nbTags(v interface{}) []string {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range list {
		switch t := item.(type) {
		case string:
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, t)
			}
		case map[string]interface{}:
			for _, k := range []string{"name", "slug"} {
				if name, ok := t[k].(string); ok && strings.TrimSpace(name) != "" {
					out = append(out, strings.TrimSpace(name))
					break
				}
			}
		}
	}
	sort.Strings(out)
	return out
}
