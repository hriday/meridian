// Package configdiff compares two device configurations line by line and
// attributes each change to the config section it falls in (the nearest
// preceding non-indented header, e.g. "interface GigabitEthernet0/1"). It is
// vendor-agnostic and works on any indentation-structured CLI config. Pure and
// offline — nothing leaves the machine.
package configdiff

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Op is the kind of a diff line.
type Op string

const (
	OpSame Op = "same"
	OpAdd  Op = "add"
	OpDel  Op = "del"
)

// Line is one line of the diff.
type Line struct {
	Op      Op     `json:"op"`
	Text    string `json:"text"`
	Section string `json:"section"` // the config section this line belongs to
}

// Result is the full diff.
type Result struct {
	Lines    []Line   `json:"lines"`
	Added    int      `json:"added"`
	Removed  int      `json:"removed"`
	Sections []string `json:"sections"` // sections containing at least one change, in order
}

// maxCells caps the LCS table size and maxDim caps either dimension; above
// either, fall back to a coarse multiset diff so a huge (or skewed) paste can't
// exhaust memory.
const (
	maxCells = 16_000_000 // ≈ 4000×4000 lines
	maxDim   = 200_000    // bounds the table even when the product is small
)

// Diff compares oldText against newText.
func Diff(oldText, newText string) Result {
	a := splitLines(oldText)
	b := splitLines(newText)
	secA := sectionsOf(a)
	secB := sectionsOf(b)

	// Fall back to the coarse diff when a side is empty (trivial all-add/all-del),
	// when either dimension is huge, or when the LCS table would be too large.
	if len(a) == 0 || len(b) == 0 ||
		len(a) > maxDim || len(b) > maxDim ||
		int64(len(a))*int64(len(b)) > maxCells {
		return coarseDiff(a, b, secA, secB)
	}
	return lcsDiff(a, b, secA, secB)
}

func lcsDiff(a, b, secA, secB []string) Result {
	n, m := len(a), len(b)
	// dp[i][j] = LCS length of a[i:] and b[j:], as a flat row-major table.
	dp := make([]int, (n+1)*(m+1))
	at := func(i, j int) int { return dp[i*(m+1)+j] }
	set := func(i, j, v int) { dp[i*(m+1)+j] = v }
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				set(i, j, at(i+1, j+1)+1)
			} else if at(i+1, j) >= at(i, j+1) {
				set(i, j, at(i+1, j))
			} else {
				set(i, j, at(i, j+1))
			}
		}
	}

	r := newResult()
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			r.add(Line{Op: OpSame, Text: a[i], Section: secA[i]})
			i++
			j++
		case at(i+1, j) >= at(i, j+1):
			r.add(Line{Op: OpDel, Text: a[i], Section: secA[i]})
			i++
		default:
			r.add(Line{Op: OpAdd, Text: b[j], Section: secB[j]})
			j++
		}
	}
	for ; i < n; i++ {
		r.add(Line{Op: OpDel, Text: a[i], Section: secA[i]})
	}
	for ; j < m; j++ {
		r.add(Line{Op: OpAdd, Text: b[j], Section: secB[j]})
	}
	return r.result()
}

// coarseDiff aligns by line multiset rather than position — approximate, but
// O(n+m) memory, used only when the inputs are too large for the LCS table.
func coarseDiff(a, b, secA, secB []string) Result {
	r := newResult()
	remB := multiset(b)
	for i, l := range a {
		if remB[l] > 0 {
			remB[l]--
			r.add(Line{Op: OpSame, Text: l, Section: secA[i]})
		} else {
			r.add(Line{Op: OpDel, Text: l, Section: secA[i]})
		}
	}
	remA := multiset(a)
	for j, l := range b {
		if remA[l] > 0 {
			remA[l]--
		} else {
			r.add(Line{Op: OpAdd, Text: l, Section: secB[j]})
		}
	}
	return r.result()
}

// resultBuilder accumulates lines, counts, and the ordered set of changed sections.
type resultBuilder struct {
	res        Result
	changedSec map[string]bool
}

func newResult() *resultBuilder {
	// Initialise the slices non-nil so the JSON sent to the UI is [] not null.
	return &resultBuilder{
		res:        Result{Lines: []Line{}, Sections: []string{}},
		changedSec: map[string]bool{},
	}
}

func (rb *resultBuilder) add(l Line) {
	rb.res.Lines = append(rb.res.Lines, l)
	switch l.Op {
	case OpAdd:
		rb.res.Added++
	case OpDel:
		rb.res.Removed++
	default:
		return
	}
	if !rb.changedSec[l.Section] {
		rb.changedSec[l.Section] = true
		rb.res.Sections = append(rb.res.Sections, l.Section)
	}
}

func (rb *resultBuilder) result() Result { return rb.res }

func multiset(lines []string) map[string]int {
	m := map[string]int{}
	for _, l := range lines {
		m[l]++
	}
	return m
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n") // "a\n" → ["a"], not ["a", ""]
	return strings.Split(s, "\n")
}

// sectionsOf labels each line with its section: the nearest preceding
// non-indented, non-blank line (a line is its own section if it is a header).
func sectionsOf(lines []string) []string {
	out := make([]string, len(lines))
	cur := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		// Blank lines and bare delimiter/comment lines (`!`, `#`) inherit the
		// current section rather than starting a new one.
		if t == "" || t[0] == '!' || t[0] == '#' {
			out[i] = cur
			continue
		}
		if !isIndented(l) {
			cur = t
		}
		out[i] = cur
	}
	return out
}

// isIndented reports whether a line begins with a whitespace rune (space, tab,
// or any Unicode space such as a non-breaking space).
func isIndented(l string) bool {
	r, _ := utf8.DecodeRuneInString(l)
	return unicode.IsSpace(r)
}
