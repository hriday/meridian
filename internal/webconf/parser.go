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
// source file and per-file line from `# configuration file …:` markers. It
// carries quote state across lines (multi-line values), skips comments and the
// `nginx -T` preamble, and consumes embedded code bodies (`*_by_lua_block`)
// opaquely so their braces/semicolons don't break the structure.
func lex(text string) []token {
	var toks []token
	file := "pasted config"
	line := 1
	var word strings.Builder
	hasWord, lineStart := false, true
	firstWord := "" // first word of the current directive, for raw-body detection
	var quote byte
	escaped := false

	emit := func() {
		if hasWord {
			s := word.String()
			if firstWord == "" {
				firstWord = s
			}
			toks = append(toks, token{text: s, file: file, line: line})
			word.Reset()
			hasWord = false
		}
	}

	for i := 0; i < len(text); {
		c := text[i]
		if quote != 0 {
			switch {
			case escaped:
				word.WriteByte(c)
				escaped = false
			case c == '\\':
				escaped = true
			case c == quote:
				quote = 0
			case c == '\n':
				word.WriteByte(c)
				line++
			default:
				word.WriteByte(c)
			}
			hasWord = true
			i++
			continue
		}
		// skip an `nginx -T` preamble line.
		if lineStart {
			lineStart = false
			k := i
			for k < len(text) && (text[k] == ' ' || text[k] == '\t') {
				k++
			}
			if strings.HasPrefix(text[k:], "nginx:") {
				for i < len(text) && text[i] != '\n' {
					i++
				}
				continue
			}
		}
		switch c {
		case '\n':
			line++
			lineStart = true
			i++
		case ' ', '\t', '\r':
			emit()
			i++
		case '"', '\'':
			quote = c
			hasWord = true
			i++
		case '#':
			j := i
			for j < len(text) && text[j] != '\n' {
				j++
			}
			trimmed := strings.TrimSpace(text[i:j])
			if strings.HasPrefix(trimmed, markerPrefix) && strings.HasSuffix(trimmed, ":") {
				file = strings.TrimSuffix(strings.TrimPrefix(trimmed, markerPrefix), ":")
				line = 1
			}
			emit()
			i = j
		case '{':
			emit()
			toks = append(toks, token{punct: '{', file: file, line: line})
			if strings.HasSuffix(firstWord, "_by_lua_block") {
				i = skipRawBody(text, i+1, &line)
				toks = append(toks, token{punct: '}', file: file, line: line})
			} else {
				i++
			}
			firstWord = ""
		case '}', ';':
			emit()
			toks = append(toks, token{punct: c, file: file, line: line})
			firstWord = ""
			i++
		default:
			word.WriteByte(c)
			hasWord = true
			i++
		}
	}
	emit()
	return toks
}

// skipRawBody consumes an embedded code body (Lua) from just past its opening
// '{' to the matching '}', honouring Lua strings, line comments and long
// brackets so their braces aren't counted. Returns the index just past the '}'.
func skipRawBody(s string, i int, line *int) int {
	depth := 1
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\n':
			*line++
			i++
		case c == '"' || c == '\'':
			i++
			for i < len(s) && s[i] != c {
				if s[i] == '\\' {
					i++
				} else if s[i] == '\n' {
					*line++
				}
				i++
			}
			i++ // closing quote
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			if i+3 < len(s) && s[i+2] == '[' && s[i+3] == '[' {
				i = skipLuaLong(s, i+4, line)
			} else {
				for i < len(s) && s[i] != '\n' {
					i++
				}
			}
		case c == '[' && i+1 < len(s) && s[i+1] == '[':
			i = skipLuaLong(s, i+2, line)
		case c == '{':
			depth++
			i++
		case c == '}':
			depth--
			i++
			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}
	return i
}

func skipLuaLong(s string, i int, line *int) int {
	for i < len(s) {
		if s[i] == '\n' {
			*line++
		}
		if s[i] == ']' && i+1 < len(s) && s[i+1] == ']' {
			return i + 2
		}
		i++
	}
	return i
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
