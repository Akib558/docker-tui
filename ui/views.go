package ui

import "strings"

// ── Top-level View ──────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	if m.dialog != dialogNone {
		return m.renderDialogOverlay()
	}

	var content string
	switch m.view {
	case viewList:
		content = m.viewList()
	case viewDetail:
		content = m.viewDetail()
	case viewImages:
		content = m.viewImages()
	case viewEvents:
		content = m.viewEvents()
	}

	// Pad to full terminal height to prevent flicker
	lines := strings.Count(content, "\n") + 1
	if lines < m.height {
		content += strings.Repeat("\n", m.height-lines)
	}
	return content
}
