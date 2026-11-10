// Package fwconfig parses firewall configuration exports (FortiGate for now)
// entirely offline and derives a structured analysis: VLANs, address/service/
// zone/policy objects, and a VLAN-level "what talks to what" flow view. It does
// no I/O and never connects to a device; encrypted secrets are never surfaced.
package fwconfig

import (
	"fmt"
	"strings"
)

// Block is a node in the parsed config tree. Kind is "root", "config" or
// "edit". Name is the section path ("system interface") or edit key
// ("port1.10"). Settings holds each `set` key's value tokens; Children are the
// nested config/edit blocks.
type Block struct {
	Kind     string
	Name     string
	Settings map[string][]string
	Children []*Block
}

func newBlock(kind, name string) *Block {
	return &Block{Kind: kind, Name: name, Settings: map[string][]string{}}
}

// parse turns FortiGate config text into a tree rooted at a synthetic "root"
// block whose Children are the top-level config blocks.
func parse(text string) (*Block, error) {
	root := newBlock("root", "")
	stack := []*Block{root}
	top := func() *Block { return stack[len(stack)-1] }

	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		toks := tokenize(line)
		if len(toks) == 0 {
			continue
		}
		switch toks[0] {
		case "config":
			b := newBlock("config", strings.Join(toks[1:], " "))
			top().Children = append(top().Children, b)
			stack = append(stack, b)
		case "edit":
			name := ""
			if len(toks) > 1 {
				name = toks[1]
			}
			b := newBlock("edit", name)
			top().Children = append(top().Children, b)
			stack = append(stack, b)
		case "next", "end":
			if len(stack) <= 1 {
				return nil, fmt.Errorf("fwconfig: unexpected %q at line %d", toks[0], i+1)
			}
			stack = stack[:len(stack)-1]
		case "set":
			if len(toks) >= 2 {
				top().Settings[toks[1]] = toks[2:]
			}
			// unset/select/unknown directives are ignored
		}
	}
	return root, nil
}

// tokenize splits a config line on whitespace, honouring double-quoted values
// with backslash-escaped quotes, and strips the surrounding quotes.
func tokenize(line string) []string {
	var toks []string
	var b strings.Builder
	inQuote, escaped, hasTok := false, false, false
	flush := func() {
		if hasTok {
			toks = append(toks, b.String())
			b.Reset()
			hasTok = false
		}
	}
	for _, r := range line {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
			hasTok = true
		case r == '\\' && inQuote:
			escaped = true
		case r == '"':
			inQuote = !inQuote
			hasTok = true // a quoted empty string is still a token
		case (r == ' ' || r == '\t') && !inQuote:
			flush()
		default:
			b.WriteRune(r)
			hasTok = true
		}
	}
	flush()
	return toks
}

// configs returns this block's direct child config blocks whose name matches
// path. Direct children only — recursion would cross VDOM scopes and mix
// objects from different virtual domains.
func (b *Block) configs(path string) []*Block {
	var out []*Block
	for _, c := range b.Children {
		if c.Kind == "config" && c.Name == path {
			out = append(out, c)
		}
	}
	return out
}

// edits returns this block's direct child edit blocks.
func (b *Block) edits() []*Block {
	var out []*Block
	for _, c := range b.Children {
		if c.Kind == "edit" {
			out = append(out, c)
		}
	}
	return out
}

// get returns a setting's value tokens joined by spaces.
func (b *Block) get(key string) (string, bool) {
	v, ok := b.Settings[key]
	if !ok {
		return "", false
	}
	return strings.Join(v, " "), true
}

// getAll returns a setting's raw value tokens.
func (b *Block) getAll(key string) []string {
	return b.Settings[key]
}
