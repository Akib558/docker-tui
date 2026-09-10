package ui

import (
	"fmt"
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
