package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

func renderTrailingDimMarker(line string, marker string, w int) string {
	if w < 1 {
		w = 1
	}
	mark := lipgloss.NewStyle().Foreground(colorDim).Render(marker)
	markW := lipgloss.Width(mark)
	if markW >= w {
		return padRenderedLine(mark, w)
	}
	contentW := w - markW
	if lipgloss.Width(line) > contentW {
		line = clampRenderedWidth(line, contentW)
	}
	return padRenderedLine(line+mark, w)
}

func padRenderedLine(s string, w int) string {
	if w < 1 {
		w = 1
	}
	if lipgloss.Width(s) > w {
		return clampRenderedWidth(s, w)
	}
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

func (m Model) logModalCountLabel() string {
	n := len(m.logModal.Entries)
	if n == 1 {
		return "1 line"
	}
	return fmt.Sprintf("%d selected", n)
}

func (m Model) logModalLeftShowsPositionHeader() bool {
	return len(m.logModal.Entries) > 1
}

func (m Model) logModalListEntryRows(totalRows int) int {
	if m.logModalLeftShowsPositionHeader() {
		return max(1, totalRows-1)
	}
	return totalRows
}

func (m Model) logModalLeftPaneActive() bool {
	return m.logModalShowsList() && m.logModal.Focus == logModalFocusLeft
}

func (m Model) logModalRightPaneActive() bool {
	return m.logModalShowsList() && m.logModal.Focus == logModalFocusRight
}

func (m Model) renderLogModalLeftPositionHeader(leftW int) string {
	label := fmt.Sprintf("%d/%d", m.logModal.Cursor+1, len(m.logModal.Entries))
	fg := colorDim
	if m.logModalLeftPaneActive() {
		fg = colorPrimary
	}
	return padRenderedLine(lipgloss.NewStyle().Foreground(fg).Bold(m.logModalLeftPaneActive()).Render(label), leftW)
}

func (m Model) logModalSourceLabel() string {
	if m.logModal.ShowCentralTag {
		return "central logs"
	}
	entry, ok := m.logModalCurrentEntry()
	if !ok {
		return ""
	}
	if entry.ContainerName != "" {
		return entry.ContainerName
	}
	return truncate(entry.ContainerID, 12)
}

func (m Model) renderLogModalTitle(innerW int) string {
	main := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Render("Log Detail · " + m.logModalCountLabel())
	source := lipgloss.NewStyle().Foreground(colorDim).Render(m.logModalSourceLabel())
	gap := innerW - lipgloss.Width(main) - lipgloss.Width(source)
	if gap < 1 {
		return clampRenderedWidth(main+" "+source, innerW)
	}
	return padRenderedLine(main+strings.Repeat(" ", gap)+source, innerW)
}

func (m Model) renderLogModalFooter() string {
	innerW, _ := m.logModalInnerSize()
	var keys []struct{ key, desc string }
	if m.logModalShowsList() {
		if m.logModalRightPaneActive() {
			keys = []struct{ key, desc string }{
				{"tab", "panes"},
				{"j/k", "scroll"},
				{"space/b", "page"},
				{"g/G", "ends"},
				{"y", "copy"},
				{"esc", "close"},
			}
		} else {
			keys = []struct{ key, desc string }{
				{"tab", "panes"},
				{"j/k", "item"},
				{"y", "copy"},
				{"Y", "all"},
				{"esc", "close"},
			}
		}
	} else {
		keys = []struct{ key, desc string }{
			{"tab", "item"},
			{"j/k", "scroll"},
			{"y", "copy"},
			{"esc", "close"},
		}
	}
	footer := fmtKeys(keys)
	for len(keys) > 2 && displayWidth(stripANSI(footer)) > innerW {
		keys = keys[:len(keys)-1]
		footer = fmtKeys(keys)
	}
	return clampRenderedWidth(footer, innerW)
}

func (m Model) renderLogDetailDialog() string {
	modalW := m.logModalWidth()
	innerW, listRows := m.logModalInnerSize()
	leftW, rightW, showList := logModalPaneWidths(innerW, m.width)

	m.logModal.ensureListScrollVisible(m.logModalListEntryRows(listRows))
	m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())

	divColor := colorBorder
	if showList {
		if m.logModalRightPaneActive() {
			divColor = colorPrimary
		}
	}
	divider := lipgloss.NewStyle().Foreground(divColor).Render("│")
	leftLines := m.logModalLeftLines(leftW, listRows)
	rightLines := m.logModalRightLines(rightW, listRows)

	var paneLines []string
	for i := 0; i < listRows; i++ {
		if showList {
			paneLines = append(paneLines, padRenderedLine(leftLines[i], leftW)+divider+padRenderedLine(rightLines[i], rightW))
		} else {
			paneLines = append(paneLines, padRenderedLine(rightLines[i], innerW))
		}
	}

	content := m.renderLogModalTitle(innerW) + "\n" +
		strings.Repeat(" ", innerW) + "\n" +
		strings.Join(paneLines, "\n") + "\n" +
		m.renderLogModalFooter()

	return dialogStyle.Width(modalW).Render(content)
}

func (m Model) logModalLeftLines(leftW int, rows int) []string {
	lines := make([]string, rows)
	if leftW < 1 {
		return lines
	}
	off := 0
	if m.logModalLeftShowsPositionHeader() {
		lines[0] = m.renderLogModalLeftPositionHeader(leftW)
		off = 1
	}
	entryRows := rows - off
	start := m.logModal.ListScroll
	for i := 0; i < entryRows; i++ {
		idx := start + i
		lineIdx := off + i
		if idx < len(m.logModal.Entries) {
			lines[lineIdx] = m.renderLogModalListRow(m.logModal.Entries[idx], leftW, idx == m.logModal.Cursor, lineIdx, m.logModalLeftPaneActive())
		} else {
			lines[lineIdx] = strings.Repeat(" ", leftW)
		}
	}
	return lines
}

func (m Model) logModalRightLines(rightW int, rows int) []string {
	lines := make([]string, rows)
	if rightW < 1 {
		return lines
	}
	entry, ok := m.logModalCurrentEntry()
	if !ok {
		for i := range lines {
			lines[i] = strings.Repeat(" ", rightW)
		}
		if rows > 0 {
			lines[0] = padRenderedLine(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("No entry"), rightW)
		}
		return lines
	}

	name := entry.ContainerName
	if name == "" {
		name = truncate(entry.ContainerID, 12)
	}
	ts := "—"
	if !entry.Timestamp.IsZero() {
		ts = entry.Timestamp.Format("2006-01-02 15:04:05.000")
	}
	level := logLevelLabel(entry.Message)
	badge := lipgloss.NewStyle().Foreground(logSeverityFG(entry.Message)).Bold(true).Render(level)

	bodyRows := max(0, rows-2)
	bodyLines := formatLogModalBodyLines(entry, rightW)
	maxScroll := maxBodyScroll(bodyLines, bodyRows)
	start := m.logModal.BodyScroll
	if start > maxScroll {
		start = maxScroll
	}

	nameFG := colorDim
	if m.logModalRightPaneActive() {
		nameFG = colorPrimary
	}
	if rows > 0 {
		nameLine := lipgloss.NewStyle().Foreground(nameFG).Bold(m.logModalRightPaneActive()).Render(name)
		lines[0] = padRenderedLine(nameLine, rightW)
	}
	if rows > 1 {
		tsLine := lipgloss.NewStyle().Foreground(colorMuted).Render(ts) + "  " + badge
		if bodyRows > 0 && len(bodyLines) > 0 {
			end := min(start+bodyRows, len(bodyLines))
			ind := lipgloss.NewStyle().Foreground(colorDim).Render(
				fmt.Sprintf("%d-%d/%d", start+1, end, len(bodyLines)))
			candidate := tsLine + "  " + ind
			if displayWidth(stripANSI(candidate)) <= rightW {
				tsLine = candidate
			} else {
				sep := "  "
				budget := rightW - displayWidth(stripANSI(ind)) - displayWidth(sep)
				if budget > 0 {
					tsLine = clampRenderedWidth(tsLine, budget) + sep + ind
				} else {
					tsLine = clampRenderedWidth(ind, rightW)
				}
			}
		}
		lines[1] = padRenderedLine(tsLine, rightW)
	}
	firstBodyRow := 2
	lastBodyRow := firstBodyRow + bodyRows - 1
	for i := 0; i < bodyRows; i++ {
		row := i + firstBodyRow
		if row >= rows {
			break
		}
		if start+i < len(bodyLines) {
			lines[row] = padRenderedLine(bodyLines[start+i], rightW)
		} else {
			lines[row] = strings.Repeat(" ", rightW)
		}
	}
	if bodyRows > 0 && start > 0 && firstBodyRow < rows {
		lines[firstBodyRow] = renderTrailingDimMarker(lines[firstBodyRow], "▲", rightW)
	}
	if bodyRows > 0 && lastBodyRow >= firstBodyRow && lastBodyRow < rows &&
		start+bodyRows < len(bodyLines) {
		lines[lastBodyRow] = renderTrailingDimMarker(lines[lastBodyRow], "▼", rightW)
	}
	for i := range lines {
		if lines[i] == "" {
			lines[i] = strings.Repeat(" ", rightW)
		}
	}
	return lines
}

func logModalEntriesSpanDays(entries []LogEntry) bool {
	if len(entries) < 2 {
		return false
	}
	var day time.Time
	seen := false
	for _, e := range entries {
		if e.Timestamp.IsZero() {
			continue
		}
		d := e.Timestamp.Truncate(24 * time.Hour)
		if !seen {
			day = d
			seen = true
			continue
		}
		if !d.Equal(day) {
			return true
		}
	}
	return false
}

func truncateMiddleDisplay(s string, maxW int) string {
	if maxW < 1 {
		return ""
	}
	if displayWidth(s) <= maxW {
		return s
	}
	const ell = "…"
	ellW := displayWidth(ell)
	if maxW <= ellW {
		return truncateDisplay(s, maxW)
	}
	inner := maxW - ellW
	if inner < 2 {
		return truncateDisplay(s, maxW)
	}
	leftW := inner / 2
	rightW := inner - leftW
	left := runewidth.Truncate(s, leftW, "")
	runes := []rune(s)
	pos := len(runes)
	w := 0
	for pos > 0 && w < rightW {
		pos--
		w += runewidth.RuneWidth(runes[pos])
	}
	right := string(runes[pos:])
	if displayWidth(left+ell+right) > maxW {
		return truncateDisplay(s, maxW)
	}
	return left + ell + right
}

func logModalEntryTagName(entry LogEntry) string {
	if entry.ContainerName != "" {
		return entry.ContainerName
	}
	return entry.ContainerID
}

func logModalTagsNeedMiddleTruncate(entries []LogEntry, tagWidth func(int) int) bool {
	truncated := make(map[string]string)
	for i, e := range entries {
		w := tagWidth(i)
		if w < 1 {
			continue
		}
		name := logModalEntryTagName(e)
		if name == "" {
			continue
		}
		t := truncateDisplay(name, w)
		if prev, ok := truncated[t]; ok && prev != name {
			return true
		}
		truncated[t] = name
	}
	return false
}

func logModalListRowStyle(focused, paneActive bool, rowIdx int) RowStyle {
	if !focused {
		return rowStyleFromKind(ListRowNormal, rowIdx)
	}
	if paneActive {
		return rowStyleFromKind(ListRowCursor, rowIdx)
	}
	return RowStyle{
		BG: colorBgSelected, Marker: glyphMarkerCursor, MarkerFG: colorDim,
		Rail: glyphRail, RailFG: colorDim, MinFG: colorSubtext,
	}
}

func (m Model) renderLogModalListRow(entry LogEntry, width int, focused bool, rowIdx int, paneActive bool) string {
	rs := logModalListRowStyle(focused, paneActive, rowIdx)
	bodyW := width - markerCols
	if bodyW < 1 {
		bodyW = 1
	}

	const levelW = 3
	level := logLevelLabel(entry.Message)
	multiDay := logModalEntriesSpanDays(m.logModal.Entries)

	timeStr := ""
	timeW := 0
	if !entry.Timestamp.IsZero() {
		timeStr = entry.Timestamp.Format("15:04:05")
		timeW = displayWidth(timeStr)
		if multiDay {
			withDate := entry.Timestamp.Format("01-02 15:04:05")
			if displayWidth(withDate)+levelW+2 <= bodyW {
				timeStr = withDate
				timeW = displayWidth(timeStr)
			}
		}
	}

	tagWidthFor := func(i int) int {
		e := m.logModal.Entries[i]
		gaps := 1
		tw := 0
		if !e.Timestamp.IsZero() {
			ts := e.Timestamp.Format("15:04:05")
			if multiDay {
				withDate := e.Timestamp.Format("01-02 15:04:05")
				if displayWidth(withDate)+levelW+2 <= bodyW {
					ts = withDate
				}
			}
			tw = displayWidth(ts)
			gaps = 2
		}
		return bodyW - tw - levelW - gaps
	}
	useMiddle := m.logModal.ShowCentralTag && logModalTagsNeedMiddleTruncate(m.logModal.Entries, tagWidthFor)

	var cells []Cell
	if timeW > 0 {
		cells = append(cells, Cell{Text: timeStr, Width: timeW, FG: colorMuted})
	}
	cells = append(cells, Cell{Text: level, Width: levelW, FG: logSeverityFG(entry.Message), Bold: true})

	if m.logModal.ShowCentralTag {
		gaps := 1
		if timeW > 0 {
			gaps = 2
		}
		tagW := bodyW - timeW - levelW - gaps
		if tagW > 0 {
			name := logModalEntryTagName(entry)
			if name == "" {
				name = truncate(entry.ContainerID, tagW)
			}
			tagText := name
			if displayWidth(name) > tagW {
				if useMiddle {
					tagText = truncateMiddleDisplay(name, tagW)
				} else {
					tagText = truncateDisplay(name, tagW)
				}
			}
			cells = append(cells, Cell{
				Text: tagText, Width: tagW,
				FG: colorTitleFg, Bold: true,
			})
		}
	}
	return RenderRow(width, cells, rs)
}
