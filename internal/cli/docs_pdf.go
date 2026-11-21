package cli

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// RenderDocsPDF lays the full tool reference out as a presentation-quality PDF:
// a branded header band, then each catalog group as a section with one row per
// command (name, usage, description), and the runbook recipes at the end.
// Generated entirely in Go (shares the fpdf helpers with the runbook PDF), so it
// works in the desktop webview where browser print-to-PDF is a no-op.
func RenderDocsPDF() ([]byte, error) {
	const (
		marginX  = 15.0
		contentW = 180.0
	)
	groups := Catalog()
	title, tools, generated := docsMeta(groups)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginX, 15, marginX)
	pdf.SetAutoPageBreak(true, 16)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pageW, pageH := pdf.GetPageSize()

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(150, 150, 155)
		pdf.CellFormat(0, 5, tr(generated), "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 5, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	// Header band.
	pdf.SetFillColor(15, 42, 46)
	pdf.Rect(0, 0, pageW, 34, "F")
	pdf.SetXY(marginX, 8)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(90, 200, 180)
	pdf.CellFormat(0, 5, tr("MERIDIAN  ·  TOOL REFERENCE"), "", 1, "L", false, 0, "")
	pdf.SetX(marginX)
	pdf.SetFont("Helvetica", "B", 19)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 10, tr(title), "", 1, "L", false, 0, "")
	pdf.SetX(marginX)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(200, 218, 214)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("%d tools across %d groups        %s", tools, len(groups), generated)), "", 1, "L", false, 0, "")

	pdf.SetY(40)
	pdf.SetX(marginX)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(110, 114, 120)
	pdf.MultiCell(contentW, 4.6, tr("Every tool is in the desktop app and as `meridian-cli <command>`. Most commands accept --json for machine-readable output."), "", "L", false)
	pdf.Ln(2)

	for _, g := range groups {
		drawDocsGroupHeading(pdf, tr, g.Name, g.Note, marginX, contentW, pageH)
		for _, c := range g.Commands {
			drawDocsCommand(pdf, tr, c.Name, c.Args, c.Desc, marginX, contentW, pageH)
		}
	}

	// Recipes.
	drawDocsGroupHeading(pdf, tr, "Runbook recipes", "meridian-cli runbook <id> <target> — composes several tools into one report", marginX, contentW, pageH)
	for _, p := range rbProfiles {
		drawDocsCommand(pdf, tr, p.id, recipeSteps(p), p.desc, marginX, contentW, pageH)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawDocsGroupHeading draws a section bar for a catalog group, keeping it off
// the very bottom of a page so it never orphans from its first command.
func drawDocsGroupHeading(pdf *fpdf.Fpdf, tr func(string) string, name, note string, marginX, contentW, pageH float64) {
	if pdf.GetY() > pageH-32 {
		pdf.AddPage()
	}
	pdf.Ln(2)
	y := pdf.GetY()
	pdf.SetFillColor(15, 42, 46)
	pdf.RoundedRect(marginX, y, contentW, 8, 1.2, "1234", "F")
	pdf.SetXY(marginX+3, y+1.4)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(contentW-6, 5, tr(name), "", 1, "L", false, 0, "")
	pdf.SetY(y + 9)
	if note != "" {
		pdf.SetX(marginX)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(120, 124, 130)
		pdf.MultiCell(contentW, 4, tr(sanitizePDF(note)), "", "L", false)
	}
	pdf.Ln(1)
}

// drawDocsCommand renders one command: its name (mono bold), usage synopsis
// (mono dim) and description (wrapped).
func drawDocsCommand(pdf *fpdf.Fpdf, tr func(string) string, name, usage, desc string, marginX, contentW, pageH float64) {
	if pdf.GetY() > pageH-22 {
		pdf.AddPage()
	}
	pdf.SetX(marginX)
	pdf.SetFont("Courier", "B", 9)
	pdf.SetTextColor(20, 50, 55)
	maxNameW := contentW * 0.45
	nameW := pdf.GetStringWidth(name) + 3
	display := name
	if nameW > maxNameW { // a very long name is truncated so it can't overrun the usage column
		nameW = maxNameW
		display = truncateToWidth(pdf, name, maxNameW-2)
	}
	pdf.CellFormat(nameW, 5, tr(display), "", 0, "L", false, 0, "")
	pdf.SetFont("Courier", "", 7.5)
	pdf.SetTextColor(130, 134, 140)
	pdf.MultiCell(contentW-nameW, 5, tr(sanitizePDF(wrapLong(usage, 70))), "", "L", false)
	if strings.TrimSpace(desc) != "" {
		pdf.SetX(marginX + 3)
		pdf.SetFont("Helvetica", "", 8.5)
		pdf.SetTextColor(70, 74, 80)
		pdf.MultiCell(contentW-3, 4.2, tr(sanitizePDF(desc)), "", "L", false)
	}
	pdf.Ln(1.6)
}

// truncateToWidth trims s (in the current font) until it plus an ASCII ellipsis
// fits within maxW mm, so a long name can't overrun its cell.
func truncateToWidth(pdf *fpdf.Fpdf, s string, maxW float64) string {
	if pdf.GetStringWidth(s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && pdf.GetStringWidth(string(r)+"...") > maxW {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}
