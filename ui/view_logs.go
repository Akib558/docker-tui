package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

func logSeverityFG(line string) lipgloss.Color {
	upper := strings.ToUpper(line)
	switch {
	case containsWord(upper, "ERROR", "ERR", "FATAL", "PANIC"):
		return colorDanger
	case containsWord(upper, "WARN", "WARNING"):
		return colorWarning
	case containsWord(upper, "DEBUG", "TRACE"):
		return colorDim
	case containsWord(upper, "INFO"):
		return colorCyan
	default:
		return colorSubtext
	}
}

func containsWord(upper string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(upper, w) {
			return true
		}
	}
	return false
}

func renderLogLegend(width int) string {
	parts := []string{
		lipgloss.NewStyle().Foreground(colorDanger).Render("ERROR"),
		lipgloss.NewStyle().Foreground(colorWarning).Render("WARN"),
		lipgloss.NewStyle().Foreground(colorCyan).Render("INFO"),
		lipgloss.NewStyle().Foreground(colorDim).Render("DEBUG"),
		lipgloss.NewStyle().Foreground(colorSecondary).Render("tag"),
	}
	line := "  " + strings.Join(parts, "  ·  ")
	return truncateDisplay(line, width)
}

func renderLogMessage(entry LogEntry, width int, showTag bool, targets map[string]LogTarget, cfg *config.Config) string {
	cells := buildLogCells(entry, width, showTag, targets, cfg)
	if len(cells) == 0 {
		return ""
	}
	var parts []string
	for _, cell := range cells {
		parts = append(parts, cellStyle(cell, "").Render(prepareCellText(cell.Text, cell.Width)))
	}
	return strings.Join(parts, "")
}

func buildLogCells(entry LogEntry, bodyW int, showTag bool, targets map[string]LogTarget, cfg *config.Config) []Cell {
	if bodyW < 20 {
		bodyW = 20
	}
	var cells []Cell
	msgBudget := bodyW

	if !entry.Timestamp.IsZero() {
		cells = append(cells, Cell{Text: entry.Timestamp.Format("15:04:05"), Width: 8, FG: colorMuted})
		msgBudget -= 9
	}
	if showTag {
		name := entry.ContainerName
		if name == "" {
			name = truncate(entry.ContainerID, 12)
		}
		tagWidth := 14
		tagColor := stableLogColor(LogTarget{ID: entry.ContainerID, Name: entry.ContainerName})
		if target, ok := targets[entry.ContainerID]; ok && target.Color != "" {
			tagColor = target.Color
		}
		cells = append(cells, Cell{
			Text:  truncate(name, tagWidth),
			Width: tagWidth,
			FG:    colorTitleFg,
			BG:    lipgloss.Color(tagColor),
			Bold:  true,
		})
		msgBudget -= tagWidth + 1
	}
	if msgBudget < 8 {
		msgBudget = 8
	}

	line := strings.ReplaceAll(entry.Message, "\n", " ")
	line = strings.ReplaceAll(line, "\t", " ")
	if lipgloss.Width(line) > msgBudget {
		line = truncate(line, msgBudget)
	}

	var msgFG lipgloss.Color
	if entry.System {
		msgFG = colorMuted
	} else {
		msgFG = logSeverityFG(line)
		if cfg != nil && len(cfg.LogHighlightPatterns) > 0 {
			for _, pattern := range cfg.LogHighlightPatterns {
				re, err := regexp.Compile(pattern.Pattern)
				if err != nil {
					continue
				}
				if re.MatchString(line) {
					msgFG = lipgloss.Color(pattern.Color)
					break
				}
			}
		}
	}

	cells = append(cells, Cell{Text: line, Width: msgBudget, FG: msgFG})
	return cells
}

func formatLogLineForCopy(entry LogEntry) string {
	var parts []string
	if !entry.Timestamp.IsZero() {
		parts = append(parts, entry.Timestamp.Format("15:04:05"))
	}
	if entry.ContainerName != "" {
		parts = append(parts, "["+entry.ContainerName+"]")
	} else if entry.ContainerID != "" {
		parts = append(parts, "["+truncate(entry.ContainerID, 12)+"]")
	}
	parts = append(parts, entry.Message)
	return strings.Join(parts, " ")
}

func formatLogLinesForCopy(entries []LogEntry) string {
	if len(entries) == 0 {
		return ""
	}
	lines := make([]string, len(entries))
	for i, entry := range entries {
		lines[i] = formatLogLineForCopy(entry)
	}
	return strings.Join(lines, "\n")
}

func containerLegendLines(targets []LogTarget, width int) int {
	if len(targets) <= 1 || width < 1 {
		return 0
	}
	lines := 1
	used := 2
	for i, t := range targets {
		item := containerLegendItem(i+1, t, false)
		itemW := lipgloss.Width(item)
		sep := 2
		if used+itemW > width {
			lines++
			used = 2 + itemW
		} else {
			used += itemW + sep
		}
	}
	return lines
}

func containerLegendItem(index int, t LogTarget, hidden bool) string {
	label := truncateDisplay(t.Name, 18)
	key := fmt.Sprintf("[%d]", index)
	if hidden {
		return lipgloss.NewStyle().Foreground(colorDim).Strikethrough(true).
			Render(key + " " + label)
	}
	return lipgloss.NewStyle().
		Foreground(colorTitleFg).
		Background(lipgloss.Color(t.Color)).
		Bold(true).
		Render(" " + key + " " + label + " ")
}

func (m Model) renderContainerLegend(width int) string {
	if len(m.centralLogTargets) <= 1 {
		return ""
	}
	var lines []string
	var current strings.Builder
	current.WriteString("  ")
	used := 2
	for i, t := range m.centralLogTargets {
		hidden := m.centralLogs.IsContainerHidden(t.ID)
		item := containerLegendItem(i+1, t, hidden)
		itemW := lipgloss.Width(item)
		if current.Len() > 2 && used+itemW > width {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString("  ")
			used = 2
		}
		if current.Len() > 2 {
			current.WriteString("  ")
			used += 2
		}
		current.WriteString(item)
		used += itemW
	}
	if current.Len() > 2 {
		lines = append(lines, current.String())
	}
	hint := lipgloss.NewStyle().Foreground(colorDim).Render("  1-9 toggle container  a show all")
	if len(lines) == 0 {
		return hint
	}
	lines = append(lines, hint)
	return strings.Join(lines, "\n")
}

func (m Model) renderLogEntryRow(entry LogEntry, width int, showTag bool, filteredIdx int, rowIdx int) string {
	lineW := width
	if lineW < markerCols+1 {
		lineW = markerCols + 1
	}
	bodyW := lineW - markerCols
	targets := m.logViewer.Targets
	cells := buildLogCells(entry, bodyW, showTag, targets, m.cfg)
	if entry.System {
		for i := range cells {
			cells[i].FG = colorMuted
			cells[i].BG = ""
		}
	}

	kind := ListRowNormal
	if m.logViewer.Selection != nil && m.logViewer.Selection[entry.Sequence] {
		kind = ListRowSelected
	}
	if filteredIdx == m.logViewer.Focused {
		if kind == ListRowSelected {
			kind = ListRowCursorSelected
		} else {
			kind = ListRowCursor
		}
	}
	return renderRowFromKind(lineW, kind, rowIdx, cells)
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
	start, end, total := m.centralLogs.VisibleWindow(m.centralLogContentRows())
	title := fmt.Sprintf("  Central Logs  %s  targets:%d", mode, len(m.centralLogTargets))
	if m.centralLogs.Filter != "" {
		if m.centralLogRegex {
			title += fmt.Sprintf("  filter(re):%q", m.centralLogs.Filter)
		} else {
			title += fmt.Sprintf("  filter:%q", m.centralLogs.Filter)
		}
	}
	if hidden := m.centralLogs.hiddenContainerKey(); hidden != "" {
		title += "  containers:filtered"
	}
	if total > 0 {
		title += fmt.Sprintf("  %d-%d/%d", start+1, end, total)
	}
	b.WriteString(sectionHeaderStyle.Width(max(m.width-4, 30)).Render(title) + "\n")
	b.WriteString(filter)
	if m.centralLogs.ShowLegend {
		b.WriteString(renderLogLegend(m.width) + "\n")
	}
	if legend := m.renderContainerLegend(m.width); legend != "" {
		b.WriteString(legend + "\n")
	}

	rows := m.centralLogs.VisibleEntries(m.centralLogContentRows())
	if len(rows) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  No logs available.") + "\n")
	} else {
		for i, entry := range rows {
			filteredIdx := start + i
			b.WriteString(m.renderCentralLogRow(entry, w, filteredIdx, i) + "\n")
		}
	}

	b.WriteString(m.renderNotification())
	b.WriteString(m.centralLogsHelp(m.width))
	return b.String()
}

func (m Model) renderCentralLogRow(entry LogEntry, width int, filteredIdx int, rowIdx int) string {
	lineW := width
	if lineW < markerCols+1 {
		lineW = markerCols + 1
	}
	bodyW := lineW - markerCols
	cells := buildLogCells(entry, bodyW, true, m.centralLogs.Targets, m.cfg)
	if entry.System {
		for i := range cells {
			cells[i].FG = colorMuted
			cells[i].BG = ""
		}
	}

	kind := ListRowNormal
	if m.centralLogs.Selection != nil && m.centralLogs.Selection[entry.Sequence] {
		kind = ListRowSelected
	}
	if filteredIdx == m.centralLogs.Focused {
		if kind == ListRowSelected {
			kind = ListRowCursorSelected
		} else {
			kind = ListRowCursor
		}
	}
	return renderRowFromKind(lineW, kind, rowIdx, cells)
}
