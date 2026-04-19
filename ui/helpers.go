package ui

import "time"

// ── Notification ────────────────────────────────────────────────────────

func (m *Model) notify(msg string, isErr bool) {
	m.notification = msg
	m.notifyIsErr = isErr
	m.notifyTime = time.Now()
}

// ── Stream lifecycle ─────────────────────────────────────────────────────

func (m *Model) stopLogStreaming() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
	m.liveLogging = false
}

func (m *Model) stopTerminalSession() {
	if m.terminalCancel != nil {
		m.terminalCancel()
		m.terminalCancel = nil
	}
	m.terminalWriter = nil
	m.terminalActive = false
}

// ── Cursor / filter helpers ──────────────────────────────────────────────

func (m *Model) clampCursorToFiltered() {
	fc := m.filteredContainers()
	if len(fc) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(fc) {
		m.cursor = len(fc) - 1
	}
}

// ── History ──────────────────────────────────────────────────────────────

func appendHist(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > historyLen {
		h = h[len(h)-historyLen:]
	}
	return h
}
