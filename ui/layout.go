package ui

import (
	"strings"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/charmbracelet/lipgloss"
)

const (
	markerCols   = 3
	rowBodyPad   = 5 // gutter(3) + right margin(2)
	tableBodyPad = 7
)

// Frame holds measured terminal layout for a view.
type Frame struct {
	TermW, TermH int
	RowW         int // highlight / row body width (termW - 5)
	TableW       int // column layout width (termW - 7)
	BodyRows     int // scrollable data rows
}

func RowW(termW int) int {
	if termW <= rowBodyPad {
		return 0
	}
	return termW - rowBodyPad
}

func TableW(termW int) int {
	if termW <= tableBodyPad {
		return 0
	}
	return termW - tableBodyPad
}

func newFrame(termW, termH, chrome int) Frame {
	return Frame{
		TermW:    termW,
		TermH:    termH,
		RowW:     RowW(termW),
		TableW:   TableW(termW),
		BodyRows: bodyRows(termH, chrome),
	}
}

func bodyRows(termH, chrome int) int {
	rows := termH - chrome
	if rows < 1 {
		return 1
	}
	return rows
}

// ListRowKind selects marker and background styling for a list row.
type ListRowKind int

const (
	ListRowNormal ListRowKind = iota
	ListRowCursor
	ListRowSelected
	ListRowCursorSelected
)

// renderSelectableRow paints a full-width row; content must be plain text.
func renderSelectableRow(lineW int, kind ListRowKind, content string) string {
	bodyW := lineW - markerCols
	if bodyW < 1 {
		bodyW = 1
	}
	return RenderRow(lineW, []Cell{{Text: content, Width: bodyW}}, rowStyleFromKind(kind, 0))
}

func (m Model) hasActiveNotification() bool {
	return m.notification != "" && since(m.notifyTime) <= 4*time.Second
}

func renderHelpBar(width int, keys string) string {
	if width < 1 {
		width = 1
	}
	return helpBarStyle.Width(width).Render(lipgloss.PlaceHorizontal(width, lipgloss.Center, keys))
}

// ── Per-view chrome (lines above + below scroll region, excluding data rows) ──

func (m Model) centralLogsFrame() Frame {
	chrome := 5 // header(2) + title(1) + gap+help(2)
	if m.centralLogFiltering {
		chrome++
	}
	if m.hasActiveNotification() {
		chrome++
	}
	return newFrame(m.width, m.height, chrome)
}

func (m Model) detailBoxInnerHeight() int {
	chrome := 11 // header(2) + identity(2) + tabs(2) + box border/pad(4) + help(1)
	if m.hasActiveNotification() {
		chrome++
	}
	h := m.height - chrome
	if h < 3 {
		return 3
	}
	return h
}

func (m Model) detailLogsChromeLines() int {
	// Section header uses BorderBottom, which renders as two terminal lines.
	lines := 2
	if m.logViewer.ShowLegend {
		lines++
	}
	return lines
}

func (m Model) detailLogContentRows() int {
	rows := m.detailBoxInnerHeight() - m.detailLogsChromeLines()
	if rows < 1 {
		return 1
	}
	return rows
}

func (m Model) centralLogsChromeLines() int {
	lines := 2
	if m.centralLogs.ShowLegend {
		lines++
	}
	if len(m.centralLogTargets) > 1 {
		lines += containerLegendLines(m.centralLogTargets, m.width)
		lines++ // toggle hint line
	}
	return lines
}

func (m Model) centralLogContentRows() int {
	rows := m.centralLogsFrame().BodyRows - m.centralLogsChromeLines()
	if rows < 1 {
		return 1
	}
	return rows
}

func clampIndex(idx, n int) int {
	if n <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= n {
		return n - 1
	}
	return idx
}

func (m *Model) onResize() {
	m.clampCursorToFiltered()

	m.imgCursor = clampIndex(m.imgCursor, len(m.images))
	m.volCursor = clampIndex(m.volCursor, len(m.volumes))
	m.netCursor = clampIndex(m.netCursor, len(m.networks))
	m.eventsCursor = clampIndex(m.eventsCursor, len(m.events))
	m.notifyCursor = clampIndex(m.notifyCursor, len(m.notifyHistory))
	m.centralLogs.normalize(m.centralLogContentRows())
	m.logViewer.normalize(m.detailLogContentRows())

	if m.commandPaletteCursor >= len(m.commandPaletteResults) {
		m.commandPaletteCursor = max(0, len(m.commandPaletteResults)-1)
	}
	if m.themeCursor >= len(config.Themes) {
		m.themeCursor = max(0, len(config.Themes)-1)
	}

	m.clampDetailScroll()
	m.syncTerminalScroll()
}

func (m *Model) clampDetailScroll() {
	if m.inspected == nil {
		m.detailScroll = 0
		return
	}
	boxWidth := max(m.width-4, 30)
	contentWidth := max(boxWidth-6, 24)
	var tabContent string
	switch m.detailTab {
	case tabInfo:
		tabContent = m.renderInfoTab(m.inspected, contentWidth)
	case tabResources:
		tabContent = m.renderResourcesTab(m.inspected, contentWidth)
	case tabEnv:
		tabContent = m.renderEnvTab(m.inspected, contentWidth)
	case tabLogs:
		tabContent = m.renderLogsTab(contentWidth)
	case tabTerminal:
		tabContent = m.renderTerminalTab(m.inspected, contentWidth)
	case tabDiff:
		tabContent = m.renderDiffTab(m.inspected, contentWidth)
	case tabProcesses:
		tabContent = m.renderProcessesTab(m.inspected, contentWidth)
	}
	lines := strings.Split(tabContent, "\n")
	avail := m.detailBoxInnerHeight()
	maxScroll := max(0, len(lines)-avail)
	if m.detailTab == tabLogs {
		m.detailScroll = 0
		return
	}
	if m.detailTab == tabTerminal {
		m.detailScroll, m.terminalFollow = normalizeTerminalScroll(m.detailScroll, maxScroll, m.terminalFollow)
		return
	}
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
	if m.detailScroll < 0 {
		m.detailScroll = 0
	}
}
