package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	padTight = 1
	padBox   = 2
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

func renderEmptyState(msg string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  "+msg) + "\n"
}

func renderLoadingState(msg string) string {
	return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  "+msg) + "\n"
}

// screenParts holds the pieces of a full-screen view.
type screenParts struct {
	header string
	title  string
	body   string
	toast  string
	help   string
}

func (m Model) renderScreen(p screenParts) string {
	var b strings.Builder
	b.WriteString(p.header)
	if p.title != "" {
		b.WriteString(p.title)
		b.WriteString("\n")
	}
	if p.body != "" {
		b.WriteString(p.body)
	}
	if p.toast == "" {
		p.toast = m.renderNotification()
	}
	if p.toast != "" {
		b.WriteString(p.toast)
	}
	if p.help != "" {
		if !strings.HasPrefix(p.help, "\n") {
			b.WriteString("\n")
		}
		b.WriteString(p.help)
	}
	return b.String()
}

func renderDialogActions(pairs []struct{ key, label string }) string {
	var parts []string
	for _, p := range pairs {
		parts = append(parts,
			lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(p.key)+
				lipgloss.NewStyle().Foreground(colorMuted).Render(" "+p.label))
	}
	return lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Join(parts, "   "))
}

func splitColumns(width, gutter int) (left, right int) {
	inner := max(width-4, 24)
	half := (inner - gutter) / 2
	if half < 12 {
		half = 12
	}
	return half, half
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
