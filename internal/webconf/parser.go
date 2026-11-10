// Package webconf parses web-server configuration exports (nginx, incl. nginx
// Plus, for now) entirely offline and derives a structured analysis: the server
// blocks, listeners, locations, upstreams and TLS, a "what routes to what"
// upstream map, and a findings list. It does no I/O and never runs the server.
package webconf

import (
	"fmt"
	"strings"
)

// Directive is a node in the parsed nginx config tree. A simple directive has
// Args and no Block; a block directive (http/server/location/upstream) has a
// Block of child directives. File/Line point at the directive's name token,
// attributed to the `# configuration file …:` section it falls in.
type Directive struct {
	Name  string
	Args  []string
	Block []*Directive
	File  string
	Line  int
}

func (d *Directive) children(name string) []*Directive {
	var out []*Directive
	for _, c := range d.Block {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

func (d *Directive) arg(i int) string {
	if i >= 0 && i < len(d.Args) {
		return d.Args[i]
	}
	return ""
}

type token struct {
	text  string // word text; empty for punctuation
	punct byte   // '{', '}', ';' — 0 for a word
	file  string
	line  int
}

const markerPrefix = "# configuration file "

// parse turns nginx config text (typically `nginx -T` output) into its
// top-level directives.
func parse(text string) ([]*Directive, error) {
	toks := lex(text)
	dirs, pos, err := buildList(toks, 0, false)
	if err != nil {
		return nil, err
	}
	if pos != len(toks) {
		return nil, fmt.Errorf("webconf: unexpected '}' at %s:%d", toks[pos].file, toks[pos].line)
	}
	return dirs, nil
}

// lex scans the config into word/punctuation tokens, tracking the current
// source file and per-file line number from `# configuration file …:` markers,
// and skipping comments and the `nginx -T` preamble.
func lex(text string) []token {
	var toks []token
	file := "pasted config"
	lineInFile := 0
	for _, raw := range strings.Split(text, "\n") {
		lineInFile++
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, markerPrefix) && strings.HasSuffix(trimmed, ":") {
			file = strings.TrimSuffix(strings.TrimPrefix(trimmed, markerPrefix), ":")
			lineInFile = 1
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "nginx:") {
			continue
		}
		lexLine(line, file, lineInFile, &toks)
	}
	return toks
}

func lexLine(line, file string, lineNo int, toks *[]token) {
	var b strings.Builder
	var quote byte // 0, '"' or '\''
	hasWord := false
	escaped := false
	flush := func() {
		if hasWord {
			*toks = append(*toks, token{text: b.String(), file: file, line: lineNo})
			b.Reset()
			hasWord = false
		}
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case escaped:
			b.WriteByte(c)
			escaped = false
			hasWord = true
		case quote != 0 && c == '\\':
			escaped = true
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				b.WriteByte(c)
			}
		case c == '"' || c == '\'':
			quote = c
			hasWord = true
		case c == '#':
			flush()
			return // rest of line is a comment
		case c == '{' || c == '}' || c == ';':
			flush()
			*toks = append(*toks, token{punct: c, file: file, line: lineNo})
		case c == ' ' || c == '\t':
			flush()
		default:
			b.WriteByte(c)
			hasWord = true
		}
	}
	flush()
}

// buildList parses directives from toks[start:] until EOF or, when inBlock, a
// closing '}'. It returns the directives and the position just past them.
func buildList(toks []token, start int, inBlock bool) ([]*Directive, int, error) {
	var out []*Directive
	i := start
	var cur *Directive
	for i < len(toks) {
		t := toks[i]
		if t.punct == 0 { // a word
			if cur == nil {
				cur = &Directive{Name: t.text, File: t.file, Line: t.line}
			} else {
				cur.Args = append(cur.Args, t.text)
			}
			i++
			continue
		}
		switch t.punct {
		case ';':
			if cur != nil {
				out = append(out, cur)
				cur = nil
			}
			i++
		case '{':
			if cur == nil {
				return nil, i, fmt.Errorf("webconf: '{' without a directive at %s:%d", t.file, t.line)
			}
			block, next, err := buildList(toks, i+1, true)
			if err != nil {
				return nil, next, err
			}
			cur.Block = block
			out = append(out, cur)
			cur = nil
			i = next
		case '}':
			if !inBlock {
				return out, i, nil // let parse() report the stray '}'
			}
			return out, i + 1, nil
		}
	}
	if inBlock {
		return nil, i, fmt.Errorf("webconf: unclosed '{'")
	}
	return out, i, nil
}
