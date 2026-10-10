package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	logModalMinWidthForList = 60
	logModalMaxOuterRows    = 32
)

// LogModalFocus is which pane receives j/k-style navigation in split layout.
type LogModalFocus int

const (
	logModalFocusLeft  LogModalFocus = 0
	logModalFocusRight LogModalFocus = 1
)

// LogModalState holds a snapshot of log lines shown in the detail modal.
type LogModalState struct {
	Entries        []LogEntry
	Targets        map[string]LogTarget
	ShowCentralTag bool
	Cursor         int
	ListScroll     int
	BodyScroll     int
	Focus          LogModalFocus
}

func logLevelLabel(line string) string {
	upper := strings.ToUpper(line)
	switch {
	case containsWord(upper, "ERROR", "ERR", "FATAL", "PANIC"):
		return "ERR"
	case containsWord(upper, "WARN", "WARNING"):
		return "WRN"
	case containsWord(upper, "DEBUG", "TRACE"):
		return "DBG"
	case containsWord(upper, "INFO"):
		return "INF"
	default:
		return "LOG"
	}
}

func logModalPaneWidths(innerW int, termW int) (leftW, rightW int, showList bool) {
	if termW < logModalMinWidthForList || innerW < 2 {
		return 0, innerW, false
	}
	leftW = innerW * 24 / 100
	if leftW < 22 {
		leftW = 22
	}
	if leftW > 40 {
		leftW = 40
	}
	rightW = innerW - leftW - 1
	if rightW < 20 {
		return 0, innerW, false
	}
	return leftW, rightW, true
}

func (m Model) logModalWidth() int {
	w := min(max(20, int(float64(m.width)*0.9)), m.width-2)
	if w < 1 {
		return 1
	}
	return w
}

func (m Model) openLogDetailModal(viewer *LogViewerState, showCentralTag bool) (Model, bool) {
	entries := viewer.SelectedEntries()
	if len(entries) == 0 {
		entry, ok := viewer.FocusedEntry()
		if !ok {
			return m, false
		}
		entries = []LogEntry{entry}
	}
	m.logModal = LogModalState{
		Entries:        append([]LogEntry(nil), entries...),
		Targets:        viewer.Targets,
		ShowCentralTag: showCentralTag,
		Focus:          logModalFocusLeft,
	}
	m.dialog = dialogLogDetail
	return m, true
}

func (m Model) logModalCurrentEntry() (LogEntry, bool) {
	if m.logModal.Cursor < 0 || m.logModal.Cursor >= len(m.logModal.Entries) {
		return LogEntry{}, false
	}
	return m.logModal.Entries[m.logModal.Cursor], true
}

func (m Model) logModalShowsList() bool {
	innerW, _ := m.logModalInnerSize()
	_, _, show := logModalPaneWidths(innerW, m.width)
	return show
}

func (m Model) logModalBodyHalfPage() int {
	step := m.logModalBodyPage() / 2
	if step < 1 {
		step = 1
	}
	return step
}

func (s *LogModalState) togglePaneFocus(backward bool) {
	if backward {
		if s.Focus == logModalFocusLeft {
			s.Focus = logModalFocusRight
		} else {
			s.Focus = logModalFocusLeft
		}
		return
	}
	if s.Focus == logModalFocusLeft {
		s.Focus = logModalFocusRight
	} else {
		s.Focus = logModalFocusLeft
	}
}

func (m Model) handleLogDetailDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	showList := m.logModalShowsList()
	rightFocused := showList && m.logModal.Focus == logModalFocusRight

	switch msg.String() {
	case "esc", "q", "enter":
		m.dialog = dialogNone
		m.logModal = LogModalState{}
		return m, nil
	case "tab":
		if showList {
			m.logModal.togglePaneFocus(false)
		} else {
			m.logModal.moveCursor(1)
		}
		return m, nil
	case "shift+tab":
		if showList {
			m.logModal.togglePaneFocus(true)
		} else {
			m.logModal.moveCursor(-1)
		}
		return m, nil
	case "left", "h":
		if showList {
			m.logModal.Focus = logModalFocusLeft
		}
		return m, nil
	case "right", "l":
		if showList {
			m.logModal.Focus = logModalFocusRight
		}
		return m, nil
	case "up", "k":
		if rightFocused {
			m.logModal.scrollBody(-1)
		} else if showList {
			m.logModal.moveCursor(-1)
			m.logModal.ensureListScrollVisible(m.logModalListEntryRows(m.logModalListRows()))
		} else {
			m.logModal.scrollBody(-1)
		}
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "down", "j":
		if rightFocused {
			m.logModal.scrollBody(1)
		} else if showList {
			m.logModal.moveCursor(1)
			m.logModal.ensureListScrollVisible(m.logModalListEntryRows(m.logModalListRows()))
		} else {
			m.logModal.scrollBody(1)
		}
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "g", "home":
		if rightFocused {
			m.logModal.BodyScroll = 0
		} else if showList {
			m.logModal.Cursor = 0
			m.logModal.BodyScroll = 0
			m.logModal.ensureListScrollVisible(m.logModalListEntryRows(m.logModalListRows()))
		} else {
			m.logModal.BodyScroll = 0
		}
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "G", "end":
		if rightFocused {
			m.logModal.BodyScroll = m.logModalMaxBodyScroll()
		} else if showList {
			if n := len(m.logModal.Entries); n > 0 {
				m.logModal.Cursor = n - 1
				m.logModal.BodyScroll = 0
				m.logModal.ensureListScrollVisible(m.logModalListEntryRows(m.logModalListRows()))
			}
		} else {
			m.logModal.BodyScroll = m.logModalMaxBodyScroll()
		}
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "pgup":
		m.logModal.scrollBody(-m.logModalBodyPage())
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "pgdown":
		m.logModal.scrollBody(m.logModalBodyPage())
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "ctrl+u":
		delta := -m.logModalBodyPage()
		if rightFocused {
			delta = -m.logModalBodyHalfPage()
		}
		m.logModal.scrollBody(delta)
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case "ctrl+d":
		delta := m.logModalBodyPage()
		if rightFocused {
			delta = m.logModalBodyHalfPage()
		}
		m.logModal.scrollBody(delta)
		m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		return m, nil
	case " ":
		if rightFocused || !showList {
			m.logModal.scrollBody(m.logModalBodyPage())
			m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		}
		return m, nil
	case "b":
		if rightFocused || !showList {
			m.logModal.scrollBody(-m.logModalBodyPage())
			m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
		}
		return m, nil
	case "y":
		entry, ok := m.logModalCurrentEntry()
		if !ok {
			return m, nil
		}
		if err := CopyLogEntries([]LogEntry{entry}); err != nil {
			m.notify("Failed to copy: "+err.Error(), true)
		} else {
			m.notify("Copied to clipboard", false)
		}
		return m, nil
	case "Y":
		if err := CopyLogEntries(m.logModal.Entries); err != nil {
			m.notify("Failed to copy: "+err.Error(), true)
		} else {
			m.notify(fmt.Sprintf("Copied %d lines", len(m.logModal.Entries)), false)
		}
		return m, nil
	default:
		return m, nil
	}
}

func (s *LogModalState) moveCursor(delta int) {
	if len(s.Entries) == 0 {
		return
	}
	s.Cursor += delta
	if s.Cursor < 0 {
		s.Cursor = 0
	}
	if s.Cursor >= len(s.Entries) {
		s.Cursor = len(s.Entries) - 1
	}
	s.BodyScroll = 0
}

func (s *LogModalState) scrollBody(delta int) {
	s.BodyScroll += delta
	if s.BodyScroll < 0 {
		s.BodyScroll = 0
	}
}

func (s *LogModalState) clampBodyScroll(maxScroll int) {
	if s.BodyScroll > maxScroll {
		s.BodyScroll = maxScroll
	}
}

func (s *LogModalState) ensureListScrollVisible(rows int) {
	if rows < 1 {
		rows = 1
	}
	maxScroll := max(0, len(s.Entries)-rows)
	if s.ListScroll > maxScroll {
		s.ListScroll = maxScroll
	}
	if s.Cursor < s.ListScroll {
		s.ListScroll = s.Cursor
	}
	if s.Cursor >= s.ListScroll+rows {
		s.ListScroll = s.Cursor - rows + 1
	}
}

func (m Model) logModalListRows() int {
	_, rows := m.logModalInnerSize()
	return rows
}

func (m Model) logModalMaxBodyScroll() int {
	entry, ok := m.logModalCurrentEntry()
	if !ok {
		return 0
	}
	innerW, listRows := m.logModalInnerSize()
	_, rightW, _ := logModalPaneWidths(innerW, m.width)
	bodyRows := max(0, listRows-2)
	lines := formatLogModalBodyLines(entry, max(1, rightW))
	return maxBodyScroll(lines, max(1, bodyRows))
}

func (m Model) logModalBodyPage() int {
	step := detailPageStep(m.height)
	if step < 3 {
		step = 3
	}
	return step
}

func (m Model) logModalInnerSize() (int, int) {
	modalW := m.logModalWidth()
	innerW := modalW - dialogStyle.GetHorizontalFrameSize()
	if innerW < 1 {
		innerW = 1
	}

	maxOuterH := m.height - 2
	if maxOuterH > logModalMaxOuterRows {
		maxOuterH = logModalMaxOuterRows
	}
	if maxOuterH < 1 {
		maxOuterH = 1
	}
	maxInnerLines := maxOuterH - dialogStyle.GetVerticalFrameSize()
	if maxInnerLines < 4 {
		maxInnerLines = 4
	}
	listRows := maxInnerLines - 3 // title, spacer, footer
	if listRows < 1 {
		listRows = 1
	}
	return innerW, listRows
}

func extractJSONPayload(msg string) (prefix string, jsonPart string, ok bool) {
	trim := strings.TrimSpace(msg)
	if json.Valid([]byte(trim)) {
		return "", trim, true
	}
	idxObj := strings.Index(msg, "{")
	idxArr := strings.Index(msg, "[")
	idx := -1
	if idxObj >= 0 && (idxArr < 0 || idxObj < idxArr) {
		idx = idxObj
	} else if idxArr >= 0 {
		idx = idxArr
	}
	if idx < 0 {
		return "", "", false
	}
	candidate := strings.TrimSpace(msg[idx:])
	if json.Valid([]byte(candidate)) {
		return strings.TrimSpace(msg[:idx]), candidate, true
	}
	return "", "", false
}

func formatLogfmtLines(msg string) ([]string, bool) {
	fields := strings.Fields(msg)
	if len(fields) < 3 {
		return nil, false
	}
	pairs := 0
	for _, f := range fields {
		if strings.Contains(f, "=") {
			pairs++
		}
	}
	if pairs < 3 {
		return nil, false
	}
	keyStyle := lipgloss.NewStyle().Foreground(colorCyan)
	lines := make([]string, 0, len(fields))
	for _, f := range fields {
		if i := strings.Index(f, "="); i > 0 {
			key := f[:i]
			val := f[i+1:]
			lines = append(lines, keyStyle.Render(key)+"="+val)
		} else {
			lines = append(lines, f)
		}
	}
	return lines, true
}

func formatLogModalBodyLines(entry LogEntry, width int) []string {
	if width < 1 {
		width = 1
	}
	msg := entry.Message
	if prefix, jsonPart, ok := extractJSONPayload(msg); ok {
		var lines []string
		if prefix != "" {
			lines = append(lines, wrapPlainPreserveBreaks(prefix, width)...)
			lines = append(lines, "")
		}
		lines = append(lines, prettyJSONLines(jsonPart, width)...)
		return lines
	}
	if logfmt, ok := formatLogfmtLines(msg); ok {
		out := make([]string, 0, len(logfmt))
		for _, line := range logfmt {
			out = append(out, wrapPlainPreserveBreaks(line, width)...)
		}
		return out
	}
	return wrapPlainPreserveBreaks(msg, width)
}

func prettyJSONLines(raw string, width int) []string {
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return wrapPlainPreserveBreaks(raw, width)
	}
	indented, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return wrapPlainPreserveBreaks(raw, width)
	}
	var out []string
	for _, line := range strings.Split(string(indented), "\n") {
		colored := colorizeJSONLine(line)
		out = append(out, wrapPlainPreserveBreaks(colored, width)...)
	}
	return out
}

func colorizeJSONLine(line string) string {
	trim := strings.TrimSpace(line)
	if !strings.HasPrefix(trim, "\"") {
		return line
	}
	end := strings.Index(trim, "\":")
	if end < 0 {
		return line
	}
	lead := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	keyPart := trim[:end+2]
	valPart := trim[end+2:]
	return lead + lipgloss.NewStyle().Foreground(colorCyan).Render(keyPart) + valPart
}

func wrapPlainPreserveBreaks(text string, width int) []string {
	if text == "" {
		return []string{""}
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		out = append(out, wrapPlainLine(line, width)...)
	}
	return out
}

func wrapPlainLine(line string, width int) []string {
	if width < 1 {
		width = 1
	}
	if displayWidth(stripANSI(line)) <= width {
		return []string{line}
	}
	var lines []string
	rest := line
	for displayWidth(stripANSI(rest)) > width {
		cut := hardWrapCut(rest, width)
		if cut <= 0 {
			break
		}
		lines = append(lines, rest[:cut])
		rest = strings.TrimLeft(rest[cut:], " ")
	}
	if rest != "" {
		lines = append(lines, rest)
	}
	if len(lines) == 0 {
		return []string{line}
	}
	return lines
}

func hardWrapCut(s string, width int) int {
	if width < 1 || s == "" {
		return 0
	}
	pos := 0
	lastSpace := -1
	for i, r := range s {
		if r == ' ' {
			lastSpace = i
		}
		rw := 1
		if pos+rw > width {
			if lastSpace > 0 {
				return lastSpace + 1
			}
			if i == 0 {
				return 1
			}
			return i
		}
		pos += rw
	}
	return len(s)
}

func maxBodyScroll(lines []string, visible int) int {
	if visible < 1 {
		visible = 1
	}
	return max(0, len(lines)-visible)
}
