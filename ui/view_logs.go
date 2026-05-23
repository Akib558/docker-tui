package ui

import "github.com/charmbracelet/lipgloss"

func (m Model) logViewportHeight() int {
	height := m.height - 15
	if height < 5 {
		return 5
	}
	return height
}

func renderLogMessage(entry LogEntry, width int, showTag bool, targets map[string]LogTarget) string {
	if width < 20 {
		width = 20
	}
	messageWidth := width - 4
	prefix := "  "
	if !entry.Timestamp.IsZero() {
		prefix += lipgloss.NewStyle().Foreground(colorMuted).Render(entry.Timestamp.Format("15:04:05")) + "  "
		messageWidth -= 10
	}
	if showTag {
		name := entry.ContainerName
		if name == "" {
			name = truncate(entry.ContainerID, 12)
		}
		tagWidth := 14
		color := stableLogColor(LogTarget{ID: entry.ContainerID, Name: entry.ContainerName})
		if target, ok := targets[entry.ContainerID]; ok && target.Color != "" {
			color = target.Color
		}
		tag := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0A0F0D")).
			Background(lipgloss.Color(color)).
			Bold(true).
			Width(tagWidth).
			Render(truncate(name, tagWidth))
		prefix += tag + "  "
		messageWidth -= tagWidth + 2
	}
	line := entry.Message
	if lipgloss.Width(line) > messageWidth {
		line = truncate(line, messageWidth)
	}
	if entry.System {
		return prefix + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(line)
	}
	return prefix + lipgloss.NewStyle().Foreground(colorSubtext).Render(line)
}
