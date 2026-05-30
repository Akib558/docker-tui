package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewVolumes() string {
	var b strings.Builder
	w := m.width

	b.WriteString(m.renderHeader(w))
	var title string
	if len(m.selected) > 0 {
		title = fmt.Sprintf("Volumes  (%d)  ◈ %d selected", len(m.volumes), len(m.selected))
	} else {
		title = fmt.Sprintf("Volumes  (%d)", len(m.volumes))
	}
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(title) + "\n\n")

	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Loading volumes...") + "\n")
		b.WriteString(m.volumesHelp(w))
		return b.String()
	}
	if len(m.volumes) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No volumes found.") + "\n")
		b.WriteString(m.volumesHelp(w))
		return b.String()
	}

	nameW := max(w*40/100, 20)
	driverW := 12
	mountW := max(w*35/100, 20)
	scopeW := 10
	usedW := nameW + driverW + mountW + scopeW + 8
	if usedW > w-4 {
		mountW = max(w-nameW-driverW-scopeW-12, 12)
	}

	hdr := "  " +
		tableHeaderStyle.Width(nameW).Render("NAME") + "  " +
		tableHeaderStyle.Width(driverW).Render("DRIVER") + "  " +
		tableHeaderStyle.Width(mountW).Render("MOUNTPOINT") + "  " +
		tableHeaderStyle.Width(scopeW).Render("SCOPE")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	visibleRows := max(3, m.height-9)
	startIdx := 0
	if m.volCursor >= visibleRows {
		startIdx = m.volCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.volumes))

	for i := startIdx; i < endIdx; i++ {
		vol := m.volumes[i]
		row := lipgloss.NewStyle().Width(nameW).Foreground(colorText).Render(truncate(vol.DisplayName(), nameW-1)) + "  " +
			lipgloss.NewStyle().Width(driverW).Foreground(colorDim).Render(truncate(vol.Driver, driverW-1)) + "  " +
			lipgloss.NewStyle().Width(mountW).Foreground(colorSubtext).Render(truncate(vol.Mountpoint, mountW-1)) + "  " +
			lipgloss.NewStyle().Width(scopeW).Foreground(colorMuted).Render(vol.Scope)
		isSelected := m.selected[vol.Name]
		rowW := w - 4
		switch {
		case i == m.volCursor && isSelected:
			mark := selectedMarkStyle.Render("◉ ")
			b.WriteString(mark + listItemSelStyle.Width(rowW).Render(row) + "\n")
		case i == m.volCursor:
			b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(rowW).Render(row) + "\n")
		case isSelected:
			mark := selectedMarkStyle.Render("◈ ")
			b.WriteString(mark + listItemStyle.Background(colorBgSelected).Width(rowW).Render(row) + "\n")
		default:
			b.WriteString("  " + listItemStyle.Width(rowW).Render(row) + "\n")
		}
	}

	if len(m.volumes) > visibleRows {
		pct := float64(m.volCursor) / float64(max(len(m.volumes)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.volCursor+1, len(m.volumes), pct)) + "\n")
	}

	b.WriteString("\n" + m.volumesHelp(w))
	return b.String()
}

func (m Model) volumesHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"space", "select"},
		{"a", "select all"},
		{"d", "remove"},
		{"p", "prune orphaned"},
		{"/", "filter"},
		{"ctrl+u", "clear filter"},
		{"r", "refresh"},
		{"t", "theme"},
		{"esc", "back"},
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}
