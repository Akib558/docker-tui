package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

func init() {
	runewidth.DefaultCondition.EastAsianWidth = false
}

// Cell is one column in a rendered row.
type Cell struct {
	Text       string
	Width      int // target display width; 0 = natural
	FG         lipgloss.Color
	BG         lipgloss.Color // per-cell background (e.g. container tag); overrides row BG
	Bold       bool
	BarPercent float64 // when Bar is true, render a mini progress bar filling Width
	Bar        bool    // render BarPercent as a bar (including 0%)
}

// RowStyle controls marker and row-level background.
type RowStyle struct {
	BG       lipgloss.Color
	Marker   string
	MarkerFG lipgloss.Color
}

func displayWidth(s string) int {
	return runewidth.StringWidth(s)
}

func truncateDisplay(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	return runewidth.Truncate(s, maxW, "")
}

// glyphPad pads text to an exact display width (fixes ambiguous-width icons).
func glyphPad(text string, targetWidth int) string {
	if targetWidth < 1 {
		return ""
	}
	w := displayWidth(text)
	if w > targetWidth {
		return truncateDisplay(text, targetWidth)
	}
	return text + strings.Repeat(" ", targetWidth-w)
}

func prepareCellText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = truncateDisplay(text, width)
	if pad := width - displayWidth(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	return text
}

func cellStyle(cell Cell, rowBG lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle()
	if cell.FG != "" {
		style = style.Foreground(cell.FG)
	}
	bg := rowBG
	if cell.BG != "" {
		bg = cell.BG
	}
	if bg != "" {
		style = style.Background(bg)
	}
	if cell.Bold {
		style = style.Bold(true)
	}
	return style
}

func renderBarSegments(percent float64, width int, rowBG lipgloss.Color) string {
	if width <= 5 {
		text := fmt.Sprintf("%3.0f%%", percent)
		return cellStyle(Cell{FG: colorSubtext}, rowBG).Render(text)
	}
	barWidth := width - 5
	if barWidth < 2 {
		barWidth = 2
	}
	barPct := percent
	if barPct > 100 {
		barPct = 100
	}
	filled := int(math.Round(barPct / 100 * float64(barWidth)))
	empty := barWidth - filled

	var fillColor lipgloss.Color
	switch {
	case percent >= 80:
		fillColor = colorDanger
	case percent >= 50:
		fillColor = colorWarning
	default:
		fillColor = colorPrimary
	}

	fillStyle := lipgloss.NewStyle().Foreground(fillColor)
	emptyStyle := lipgloss.NewStyle().Foreground(colorBarTrack)
	pctStyle := lipgloss.NewStyle().Foreground(colorSubtext)
	if rowBG != "" {
		fillStyle = fillStyle.Background(rowBG)
		emptyStyle = emptyStyle.Background(rowBG)
		pctStyle = pctStyle.Background(rowBG)
	}

	bar := fillStyle.Render(strings.Repeat(glyphBarFill, filled)) +
		emptyStyle.Render(strings.Repeat(glyphBarEmpty, empty))
	out := bar + pctStyle.Render(" "+fmt.Sprintf("%3.0f%%", percent))
	if got := lipgloss.Width(out); got > width {
		out = clampRenderedWidth(out, width)
	}
	return out
}

// RenderRow paints a full-width row with background-aware cells.
func RenderRow(totalWidth int, cells []Cell, rs RowStyle) string {
	if totalWidth < 1 {
		totalWidth = 1
	}
	cells = fitCells(totalWidth, cells)
	bg := rs.BG

	marker := rs.Marker
	if marker == "" {
		marker = " "
	}
	markerStyle := lipgloss.NewStyle()
	if rs.MarkerFG != "" {
		markerStyle = markerStyle.Foreground(rs.MarkerFG).Bold(true)
	}
	if bg != "" {
		markerStyle = markerStyle.Background(bg)
	}
	line := markerStyle.Render(glyphPad(marker, 1)) + " "

	for i, cell := range cells {
		var part string
		if cell.Bar && cell.Width > 0 {
			part = renderBarSegments(cell.BarPercent, cell.Width, bg)
		} else {
			text := prepareCellText(cell.Text, cell.Width)
			part = cellStyle(cell, bg).Render(text)
		}
		line += part
		if i < len(cells)-1 {
			spaceStyle := lipgloss.NewStyle()
			if bg != "" {
				spaceStyle = spaceStyle.Background(bg)
			}
			line += spaceStyle.Render(" ")
		}
	}

	if pad := totalWidth - lipgloss.Width(line); pad > 0 {
		padStyle := lipgloss.NewStyle()
		if bg != "" {
			padStyle = padStyle.Background(bg)
		}
		line += padStyle.Render(strings.Repeat(" ", pad))
	}
	if lipgloss.Width(line) > totalWidth {
		line = clampRenderedWidth(line, totalWidth)
	}
	return line
}

func clampRenderedWidth(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	return ansi.Truncate(s, maxW, "")
}

func rowStyleFromKind(kind ListRowKind, rowIdx int) RowStyle {
	switch kind {
	case ListRowCursor:
		return RowStyle{BG: colorBgSelected, Marker: glyphMarkerCursor, MarkerFG: colorPrimary}
	case ListRowSelected:
		return RowStyle{BG: colorBgSelected, Marker: glyphMarkerSelect, MarkerFG: colorWarning}
	case ListRowCursorSelected:
		return RowStyle{BG: colorBgSelected, Marker: glyphMarkerBoth, MarkerFG: colorPrimary}
	default:
		rs := RowStyle{Marker: " "}
		if rowIdx%2 == 1 {
			rs.BG = colorBgAlt
		}
		return rs
	}
}

func stateFGColor(state string) lipgloss.Color {
	switch state {
	case "running":
		return colorSuccess
	case "exited", "dead":
		return colorDanger
	case "paused", "restarting":
		return colorWarning
	case "created":
		return colorMuted
	default:
		return colorMuted
	}
}

func healthIconPlain(health string) string { return healthGlyphPlain(health) }

func renderRowFromKind(termW int, kind ListRowKind, rowIdx int, cells []Cell) string {
	return RenderRow(termW, cells, rowStyleFromKind(kind, rowIdx))
}

func fitCells(totalWidth int, cells []Cell) []Cell {
	if len(cells) == 0 {
		return cells
	}
	avail := totalWidth - markerCols
	if len(cells) > 1 {
		avail -= len(cells) - 1
	}
	if avail < 1 {
		avail = 1
	}
	out := append([]Cell(nil), cells...)
	for cellTotal(out) > avail && len(out) > 0 {
		shrunk := false
		for i := len(out) - 1; i >= 0; i-- {
			if out[i].Width > 1 {
				out[i].Width--
				shrunk = true
				break
			}
		}
		if !shrunk {
			out = out[:len(out)-1]
		}
	}
	return out
}

func cellTotal(cells []Cell) int {
	total := 0
	for _, c := range cells {
		if c.Width > 0 {
			total += c.Width
		}
	}
	return total
}
