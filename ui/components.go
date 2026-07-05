package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ListColumn describes one table column header.
type ListColumn struct {
	Label string
	Width int
}

// ListTableRow is one data row in a ListTable.
type ListTableRow struct {
	Cells    []Cell
	Selected bool
}

// ListTable renders a width-safe table with marker gutter, zebra, and cursor.
func ListTable(termW int, cols []ListColumn, rows []ListTableRow, cursor int, visibleRows int) string {
	if len(cols) == 0 {
		return ""
	}
	var b strings.Builder
	headerCells := make([]Cell, len(cols))
	for i, c := range cols {
		headerCells[i] = Cell{Text: c.Label, Width: c.Width, FG: colorMuted, Bold: true}
	}
	b.WriteString(RenderRow(termW, headerCells, RowStyle{Marker: " "}) + "\n")

	start := 0
	if cursor >= visibleRows {
		start = cursor - visibleRows + 1
	}
	end := min(start+visibleRows, len(rows))
	for i := start; i < end; i++ {
		row := rows[i]
		kind := listRowKind(i, cursor, row.Selected)
		b.WriteString(renderRowFromKind(termW, kind, i, row.Cells) + "\n")
	}
	if len(rows) > visibleRows {
		pct := float64(cursor) / float64(max(len(rows)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  %s %d/%d (%.0f%%)", glyphScroll, cursor+1, len(rows), pct)) + "\n")
	}
	return b.String()
}

func listRowKind(idx, cursor int, selected bool) ListRowKind {
	switch {
	case idx == cursor && selected:
		return ListRowCursorSelected
	case idx == cursor:
		return ListRowCursor
	case selected:
		return ListRowSelected
	default:
		return ListRowNormal
	}
}

func renderViewTitle(name string, count int, selected int) string {
	title := fmt.Sprintf("%s  (%d)", name, count)
	if selected > 0 {
		title += fmt.Sprintf("  %s %d selected", glyphMarkerSelect, selected)
	}
	return titleStyle.Render(title) + "\n"
}

// StateBadge returns a width-padded state glyph, optionally with health mark.
func StateBadge(state, health string, width int) string {
	icon := stateGlyph(state)
	if health != "" {
		icon += healthGlyphPlain(health)
	}
	return glyphPad(icon, width)
}

// StateBadgeStyled returns a styled state label for detail headers.
func StateBadgeStyled(state string) string {
	return stateStyle(state).Render(stateGlyph(state) + " " + stateDisplayName(state))
}
