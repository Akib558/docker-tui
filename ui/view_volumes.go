package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewVolumes() string {
	var b strings.Builder
	w := m.width
	vols := m.filteredVolumes()

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
	if len(vols) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No volumes match filter.") + "\n")
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

	frame := m.volumesFrame()
	visibleRows := frame.BodyRows
	startIdx := 0
	if m.volCursor >= visibleRows {
		startIdx = m.volCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(vols))

	for i := startIdx; i < endIdx; i++ {
		vol := vols[i]
		cells := []Cell{
			{Text: truncate(vol.DisplayName(), nameW), Width: nameW, FG: colorText},
			{Text: truncate(vol.Driver, driverW), Width: driverW, FG: colorDim},
			{Text: truncate(vol.Mountpoint, mountW), Width: mountW, FG: colorSubtext},
			{Text: vol.Scope, Width: scopeW, FG: colorMuted},
		}
		isSelected := m.selected[vol.Name]
		var kind ListRowKind
		switch {
		case i == m.volCursor && isSelected:
			kind = ListRowCursorSelected
		case i == m.volCursor:
			kind = ListRowCursor
		case isSelected:
			kind = ListRowSelected
		default:
			kind = ListRowNormal
		}
		b.WriteString(renderRowFromKind(w, kind, i, cells) + "\n")
	}

	if len(vols) > visibleRows {
		pct := float64(m.volCursor) / float64(max(len(vols)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.volCursor+1, len(vols), pct)) + "\n")
	}

	b.WriteString("\n" + m.volumesHelp(w))
	return b.String()
}

func (m Model) volumesHelp(w int) string {
	if m.volFiltering {
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
		{"p", "prune"},
		{"/", "filter"},
		{"?", "help"},
		{"esc", "back"},
	}
	return renderHelpBar(w, fmtKeys(keys))
}
