package cli

import (
	"bytes"
	"strings"

	"github.com/go-pdf/fpdf"
)

// RenderRunbookPDF lays an already-run runbook out as a PDF — a title block then
// each step's command (graded) and its output in a monospace font. It is
// generated entirely in Go so it works in the desktop webview, where the browser
// print-to-PDF path is a no-op (macOS WKWebView doesn't implement window.print).
func RenderRunbookPDF(d RunbookData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	tr := pdf.UnicodeTranslatorFromDescriptor("") // UTF-8 -> the core font's encoding
	pdf.AddPage()

	name := d.Name
	if name == "" {
		name = d.Profile
	}
	pdf.SetFont("Helvetica", "B", 16)
	pdf.MultiCell(0, 8, tr("Meridian Runbook — "+name), "", "L", false)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(90, 90, 90)
	if d.Target != "" {
		pdf.MultiCell(0, 5, tr("Target: "+d.Target), "", "L", false)
	}
	if d.Generated != "" {
		pdf.MultiCell(0, 5, tr("Generated: "+d.Generated), "", "L", false)
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)

	for _, s := range d.Steps {
		cmd := s.Command
		if len(s.Args) > 0 {
			cmd += " " + strings.Join(s.Args, " ")
		}
		status := "ok"
		if !s.OK {
			status = "failed"
		}
		pdf.SetFont("Helvetica", "B", 11)
		if s.OK {
			pdf.SetTextColor(20, 110, 40)
		} else {
			pdf.SetTextColor(170, 40, 40)
		}
		pdf.MultiCell(0, 6, tr(cmd+"  ["+status+"]"), "", "L", false)
		pdf.SetTextColor(0, 0, 0)

		out := wrapLong(sanitizePDF(s.Output), 100)
		if strings.TrimSpace(out) == "" {
			out = "(no output)"
		}
		pdf.SetFont("Courier", "", 8)
		pdf.MultiCell(0, 3.8, tr(out), "", "L", false)
		pdf.Ln(2.5)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sanitizePDF replaces common non-latin-1 glyphs (box drawing, check marks,
// arrows) with ASCII so the core PDF fonts render them legibly.
func sanitizePDF(s string) string {
	return strings.NewReplacer(
		"✓", "[ok]", "✔", "[ok]", "✗", "[x]", "✘", "[x]", "×", "x",
		"•", "-", "·", "-", "→", "->", "←", "<-", "…", "...",
		"─", "-", "│", "|", "├", "|", "└", "|", "┌", "|", "┐", "|", "┘", "|",
		"⚠", "(!)", "°", " deg", "’", "'", "“", "\"", "”", "\"",
	).Replace(s)
}

// wrapLong hard-wraps any line longer than max runes, so a long unbroken token
// (a hash, a base64 blob) can't overflow the PDF cell width.
func wrapLong(s string, max int) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		r := []rune(line)
		for len(r) > max {
			out = append(out, string(r[:max]))
			r = r[max:]
		}
		out = append(out, string(r))
	}
	return strings.Join(out, "\n")
}
