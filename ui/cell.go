package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/akib558/docker-tui/config"
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

// RowStyle controls marker, rail and row-level background.
type RowStyle struct {
	BG       lipgloss.Color
	Marker   string
	MarkerFG lipgloss.Color
	Rail     string // left accent glyph; " " when the row has no accent
	RailFG   lipgloss.Color
	MinFG    lipgloss.Color // floor colour for cells too close to BG to read
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

func cellFG(cell Cell, rowBG, minFG lipgloss.Color) (fg, bg lipgloss.Color, bold bool) {
	bg = rowBG
	if cell.BG != "" {
		bg = cell.BG
	}
	fg = cell.FG
	if fg != "" && bg != "" && minFG != "" && !readableOn(fg, bg) {
		fg = minFG
	}
	return fg, bg, cell.Bold
}

func renderCellPart(cell Cell, rowBG, minFG lipgloss.Color) string {
	if cell.Bar && cell.Width > 0 {
		return renderBarSegments(cell.BarPercent, cell.Width, rowBG)
	}
	fg, bg, bold := cellFG(cell, rowBG, minFG)
	text := prepareCellText(cell.Text, cell.Width)
	return wrapWithStyle(text, fg, bg, bold)
}

// readableOn reports whether fg is distinguishable from bg.
func readableOn(fg, bg lipgloss.Color) bool {
	ratio, ok := config.ContrastRatio(string(fg), string(bg))
	if !ok {
		return true // non-hex colours (named/ANSI) are left alone
	}
	return ratio >= 1.6
}

func renderBarSegments(percent float64, width int, rowBG lipgloss.Color) string {
	if width <= 5 {
		text := fmt.Sprintf("%3.0f%%", percent)
		return wrapWithStyle(text, colorSubtext, rowBG, false)
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

	emptyFG := colorBarTrack
	if rowBG != "" && !readableOn(colorBarTrack, rowBG) {
		emptyFG = colorSubtext
	}
	bar := wrapWithStyle(strings.Repeat(glyphBarFill, filled), fillColor, rowBG, false) +
		wrapWithStyle(strings.Repeat(glyphBarEmpty, empty), emptyFG, rowBG, false)
	out := bar + wrapWithStyle(" "+fmt.Sprintf("%3.0f%%", percent), colorSubtext, rowBG, false)
	if got := renderedWidth(out); got > width {
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

	rail := rs.Rail
	if rail == "" {
		rail = " "
	}
	marker := rs.Marker
	if marker == "" {
		marker = " "
	}

	var b strings.Builder
	b.Grow(totalWidth * 4)
	b.WriteString(wrapWithStyle(glyphPad(rail, 1), rs.RailFG, bg, rs.RailFG != ""))
	b.WriteString(wrapWithStyle(glyphPad(marker, 1), rs.MarkerFG, bg, rs.MarkerFG != ""))
	b.WriteString(bgPad(1, bg))

	for i, cell := range cells {
		b.WriteString(renderCellPart(cell, bg, rs.MinFG))
		if i < len(cells)-1 {
			b.WriteString(bgPad(1, bg))
		}
	}

	line := b.String()
	if pad := totalWidth - renderedWidth(line); pad > 0 {
		line += bgPad(pad, bg)
	}
	if renderedWidth(line) > totalWidth {
		line = clampRenderedWidth(line, totalWidth)
	}
	return line
}

func clampRenderedWidth(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	if renderedWidth(s) <= maxW {
		return s
	}
	return ansi.Truncate(s, maxW, "")
}

func rowStyleFromKind(kind ListRowKind, rowIdx int) RowStyle {
	switch kind {
	case ListRowCursor:
		return RowStyle{BG: colorBgSelected, Marker: glyphMarkerCursor, MarkerFG: colorPrimary,
			Rail: glyphRail, RailFG: colorPrimary, MinFG: colorSubtext}
	case ListRowSelected:
		return RowStyle{BG: colorBgSelected, Marker: glyphMarkerSelect, MarkerFG: colorWarning,
			Rail: glyphRail, RailFG: colorWarning, MinFG: colorSubtext}
	case ListRowCursorSelected:
		return RowStyle{BG: colorBgSelected, Marker: glyphMarkerBoth, MarkerFG: colorPrimary,
			Rail: glyphRail, RailFG: colorPrimary, MinFG: colorSubtext}
	default:
		rs := RowStyle{Marker: " "}
		if rowIdx%2 == 1 {
			rs.BG = colorBgAlt
			rs.MinFG = colorSubtext
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
