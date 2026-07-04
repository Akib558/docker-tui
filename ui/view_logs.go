package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) logViewportHeight() int {
	height := m.height - 15
	if height < 5 {
		return 5
	}
	return height
}

func (m Model) detailLogContentRows() int {
	rows := m.logViewportHeight() - 1
	if rows < 1 {
		return 1
	}
	return rows
}

func renderLogMessage(entry LogEntry, width int, showTag bool, targets map[string]LogTarget, cfg *config.Config) string {
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

	// Apply log highlighting
	if cfg != nil && len(cfg.LogHighlightPatterns) > 0 {
		highlighted := highlightLogLine(line, cfg.LogHighlightPatterns)
		return prefix + highlighted
	}

	return prefix + lipgloss.NewStyle().Foreground(colorSubtext).Render(line)
}

func highlightLogLine(line string, patterns []config.LogHighlightPattern) string {
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			continue
		}
		if re.MatchString(line) {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(pattern.Color)).Bold(true).Render(line)
		}
	}
	return lipgloss.NewStyle().Foreground(colorSubtext).Render(line)
}

func (m Model) viewCentralLogs() string {
	var b strings.Builder
	w := m.width
	if w < 30 {
		w = 30
	}
	b.WriteString(m.renderHeader(m.width))

	filter := ""
	if m.centralLogFiltering {
		text := m.centralLogFilter
		if text == "" {
			text = "type to filter logs..."
		}
		if m.centralLogRegex {
			text += " (regex)"
		}
		filter = "  " + filterBarStyle.Render("⌕ "+text) + "\n"
	}

	mode := "FOLLOW"
	if !m.centralLogs.Follow {
		mode = "PAUSED"
	}
	start, end, total := m.centralLogs.VisibleWindow(m.centralLogViewportHeight())
	title := fmt.Sprintf("  Central Logs  %s  targets:%d", mode, len(m.centralLogTargets))
	if m.centralLogs.Filter != "" {
		if m.centralLogRegex {
			title += fmt.Sprintf("  filter(re):%q", m.centralLogs.Filter)
		} else {
			title += fmt.Sprintf("  filter:%q", m.centralLogs.Filter)
		}
	}
	if total > 0 {
		title += fmt.Sprintf("  %d-%d/%d", start+1, end, total)
	}
	b.WriteString(sectionHeaderStyle.Width(max(m.width-4, 30)).Render(title) + "\n")
	b.WriteString(filter)

	rows := m.centralLogs.VisibleEntries(m.centralLogViewportHeight())
	if len(rows) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  No logs available.") + "\n")
	} else {
		for i, entry := range rows {
			line := renderLogMessage(entry, w-4, true, m.centralLogs.Targets, m.cfg)
			if i == m.centralLogCursor {
				b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(w-4).Render(line) + "\n")
			} else {
				b.WriteString("  " + line + "\n")
			}
		}
	}

	b.WriteString(m.renderNotification())
	b.WriteString(m.centralLogsHelp(m.width))
	return b.String()
}

func (m Model) centralLogViewportHeight() int {
	height := m.height - 8
	if m.centralLogFiltering {
		height--
	}
	if height < 5 {
		return 5
	}
	return height
}
