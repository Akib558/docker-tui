package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewNetworks() string {
	var b strings.Builder
	w := m.width
	nets := m.filteredNetworks()

	b.WriteString(m.renderHeader(w))
	var title string
	if len(m.selected) > 0 {
		title = fmt.Sprintf("Networks  (%d)  ◈ %d selected", len(m.networks), len(m.selected))
	} else {
		title = fmt.Sprintf("Networks  (%d)", len(m.networks))
	}
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(title) + "\n\n")

	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Loading networks...") + "\n")
		b.WriteString(m.networksHelp(w))
		return b.String()
	}
	if len(m.networks) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No networks found.") + "\n")
		b.WriteString(m.networksHelp(w))
		return b.String()
	}
	if len(nets) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No networks match filter.") + "\n")
		b.WriteString(m.networksHelp(w))
		return b.String()
	}

	nameW := max(w*30/100, 18)
	driverW := 12
	scopeW := 10
	internalW := 10
	usedW := nameW + driverW + scopeW + internalW + 10
	if usedW > w-4 {
		nameW = max(w-driverW-scopeW-internalW-14, 12)
	}

	hdr := "  " +
		tableHeaderStyle.Width(nameW).Render("NAME") + "  " +
		tableHeaderStyle.Width(driverW).Render("DRIVER") + "  " +
		tableHeaderStyle.Width(scopeW).Render("SCOPE") + "  " +
		tableHeaderStyle.Width(internalW).Render("INTERNAL")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	frame := m.networksFrame()
	visibleRows := frame.BodyRows
	startIdx := 0
	if m.netCursor >= visibleRows {
		startIdx = m.netCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(nets))

	for i := startIdx; i < endIdx; i++ {
		net := nets[i]
		internalStr := "no"
		internalFG := colorMuted
		if net.Internal {
			internalStr = "yes"
			internalFG = colorSuccess
		}
		cells := []Cell{
			{Text: truncate(net.Name, nameW), Width: nameW, FG: colorText},
			{Text: truncate(net.Driver, driverW), Width: driverW, FG: colorDim},
			{Text: net.Scope, Width: scopeW, FG: colorSubtext},
			{Text: internalStr, Width: internalW, FG: internalFG},
		}
		isSelected := m.selected[net.ID]
		var kind ListRowKind
		switch {
		case i == m.netCursor && isSelected:
			kind = ListRowCursorSelected
		case i == m.netCursor:
			kind = ListRowCursor
		case isSelected:
			kind = ListRowSelected
		default:
			kind = ListRowNormal
		}
		b.WriteString(renderRowFromKind(w, kind, i, cells) + "\n")
	}

	if len(nets) > visibleRows {
		pct := float64(m.netCursor) / float64(max(len(nets)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.netCursor+1, len(nets), pct)) + "\n")
	}

	b.WriteString("\n" + m.networksHelp(w))
	return b.String()
}

func (m Model) networksHelp(w int) string {
	if m.netFiltering {
		return renderHelpBar(w, fmtKeys([]struct{ key, desc string }{
			{"type", "search"},
			{"backspace", "delete"},
			{"enter/esc", "done"},
			{"ctrl+u", "clear"},
		}))
	}
	keys := []struct{ key, desc string }{
		{"j/k", "nav"},
		{"space", "select"},
		{"d", "remove"},
		{"/", "filter"},
		{"?", "help"},
		{"esc", "back"},
	}
	return renderHelpBar(w, fmtKeys(keys))
}
