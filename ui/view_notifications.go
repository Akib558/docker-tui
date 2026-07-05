package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewNotifications() string {
	var b strings.Builder
	w := m.width

	b.WriteString(m.renderHeader(w))
	var title string
	if len(m.notifyHistory) == 0 {
		title = "Notifications  (empty)"
	} else {
		title = fmt.Sprintf("Notifications  (%d)", len(m.notifyHistory))
	}
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(title) + "\n\n")

	if len(m.notifyHistory) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No notifications yet.") + "\n")
		b.WriteString(m.notificationsHelp(w))
		return b.String()
	}

	frame := m.notificationsFrame()
	visibleRows := frame.BodyRows
	startIdx := 0
	if m.notifyCursor >= visibleRows {
		startIdx = m.notifyCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.notifyHistory))

	for i := startIdx; i < endIdx; i++ {
		notif := m.notifyHistory[i]
		icon := "✓"
		fg := colorSuccess
		if notif.IsError {
			icon = "✗"
			fg = colorDanger
		}
		timeStr := notif.Timestamp.Format("15:04:05")
		msg := truncate(notif.Message, max(w-30, 20))
		cells := []Cell{
			{Text: timeStr, Width: 8, FG: colorMuted},
			{Text: icon, Width: 1, FG: fg},
			{Text: msg, Width: max(w-14, 20), FG: fg},
		}
		kind := ListRowNormal
		if i == m.notifyCursor {
			kind = ListRowCursor
		}
		b.WriteString(renderRowFromKind(w, kind, i, cells) + "\n")
	}

	if len(m.notifyHistory) > visibleRows {
		pct := float64(m.notifyCursor) / float64(max(len(m.notifyHistory)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.notifyCursor+1, len(m.notifyHistory), pct)) + "\n")
	}

	b.WriteString("\n" + m.notificationsHelp(w))
	return b.String()
}

func (m Model) notificationsHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"c", "clear all"},
		{"esc", "back"},
	}
	return renderHelpBar(w, fmtKeys(keys))
}
