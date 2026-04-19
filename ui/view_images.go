package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewImages() string {
	var b strings.Builder
	w := m.width

	b.WriteString(m.renderHeader(w))
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Render(fmt.Sprintf("Images  (%d)", len(m.images))) + "\n\n")

	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Loading images...") + "\n")
		b.WriteString(m.imagesHelp(w))
		return b.String()
	}
	if len(m.images) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No images found.") + "\n")
		b.WriteString(m.imagesHelp(w))
		return b.String()
	}

	tagW := max(w*35/100, 20)
	idW := 14
	sizeW := 10
	dateW := 16
	usedW := tagW + idW + sizeW + dateW + 8
	if usedW > w-4 {
		tagW = max(w-idW-sizeW-dateW-12, 12)
	}

	hdr := "  " +
		tableHeaderStyle.Width(tagW).Render("TAG") + "  " +
		tableHeaderStyle.Width(idW).Render("IMAGE ID") + "  " +
		tableHeaderStyle.Width(sizeW).Render("SIZE") + "  " +
		tableHeaderStyle.Width(dateW).Render("CREATED")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	usedLines := 9
	visibleRows := max(3, m.height-usedLines)
	startIdx := 0
	if m.imgCursor >= visibleRows {
		startIdx = m.imgCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.images))

	for i := startIdx; i < endIdx; i++ {
		img := m.images[i]
		row := lipgloss.NewStyle().Width(tagW).Foreground(colorText).Render(truncate(img.DisplayTag(), tagW-1)) + "  " +
			lipgloss.NewStyle().Width(idW).Foreground(colorDim).Render(truncate(img.ID, idW-1)) + "  " +
			lipgloss.NewStyle().Width(sizeW).Foreground(colorSubtext).Render(formatBytes(uint64(img.Size))) + "  " +
			lipgloss.NewStyle().Width(dateW).Foreground(colorMuted).Render(img.Created.Format("2006-01-02 15:04"))
		if i == m.imgCursor {
			b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(w-4).Render(row) + "\n")
		} else {
			b.WriteString("  " + listItemStyle.Width(w-4).Render(row) + "\n")
		}
	}

	if len(m.images) > visibleRows {
		pct := float64(m.imgCursor) / float64(max(len(m.images)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.imgCursor+1, len(m.images), pct)) + "\n")
	}

	b.WriteString("\n" + m.imagesHelp(w))
	return b.String()
}

func (m Model) imagesHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"p", "pull image"},
		{"d", "remove"},
		{"r", "refresh"},
		{"t", "theme"},
		{"esc", "back"},
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}
