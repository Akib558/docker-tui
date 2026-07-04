package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewNetworks() string {
	var b strings.Builder
	w := m.width

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

	nameW := max(w*30/100, 18)
	driverW := 12
	scopeW := 10
	internalW := 10
	usedW := nameW + driverW + scopeW + internalW + 10
	if usedW > w-4 {
		nameW = max(w-nameW-driverW-scopeW-internalW-14, 12)
	}

	hdr := "  " +
		tableHeaderStyle.Width(nameW).Render("NAME") + "  " +
		tableHeaderStyle.Width(driverW).Render("DRIVER") + "  " +
		tableHeaderStyle.Width(scopeW).Render("SCOPE") + "  " +
		tableHeaderStyle.Width(internalW).Render("INTERNAL")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	visibleRows := max(3, m.height-9)
	startIdx := 0
	if m.netCursor >= visibleRows {
		startIdx = m.netCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.networks))

	for i := startIdx; i < endIdx; i++ {
		net := m.networks[i]
		internalStr := ""
		if net.Internal {
			internalStr = lipgloss.NewStyle().Foreground(colorSuccess).Render("yes")
		} else {
			internalStr = lipgloss.NewStyle().Foreground(colorMuted).Render("no")
		}
		row := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(net.Name, nameW-1)) + "  " +
			lipgloss.NewStyle().Width(driverW).Foreground(colorDim).Render(truncate(net.Driver, driverW-1)) + "  " +
			lipgloss.NewStyle().Width(scopeW).Foreground(colorSubtext).Render(net.Scope) + "  " +
			lipgloss.NewStyle().Width(internalW).Render(internalStr)
		isSelected := m.selected[net.ID]
		rowW := w - 4
		switch {
		case i == m.netCursor && isSelected:
			mark := selectedMarkStyle.Render("◉ ")
			b.WriteString(mark + listItemSelStyle.Width(rowW).Render(row) + "\n")
		case i == m.netCursor:
			b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(rowW).Render(row) + "\n")
		case isSelected:
			mark := selectedMarkStyle.Render("◈ ")
			b.WriteString(mark + listItemStyle.Background(colorBgSelected).Width(rowW).Render(row) + "\n")
		default:
			b.WriteString("  " + listItemStyle.Width(rowW).Render(row) + "\n")
		}
	}

	if len(m.networks) > visibleRows {
		pct := float64(m.netCursor) / float64(max(len(m.networks)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.netCursor+1, len(m.networks), pct)) + "\n")
	}

	b.WriteString("\n" + m.networksHelp(w))
	return b.String()
}

func (m Model) networksHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "nav"},
		{"space", "select"},
		{"d", "remove"},
		{"/", "filter"},
		{"?", "help"},
		{"esc", "back"},
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}
