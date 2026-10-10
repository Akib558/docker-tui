package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/akib558/docker-tui/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func logModalTestEntry() LogEntry {
	return LogEntry{
		ContainerName: "api",
		Message:       `{"msg":"db timeout","level":"error"}`,
		Timestamp:     time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC),
		Sequence:      1,
	}
}

func modelWithLogModal(width, height int, entries []LogEntry, showCentral bool) Model {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = width
	m.height = height
	m.dialog = dialogLogDetail
	m.logModal = LogModalState{
		Entries:        append([]LogEntry(nil), entries...),
		ShowCentralTag: showCentral,
	}
	return m
}

func TestLogModalOpensFromDetailLogsTab(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.view = viewDetail
	m.detailTab = tabLogs
	m.width = 80
	m.height = 30
	m.logViewer = NewLogViewerState(detailLogBufferMax, nil)
	m.logViewer.Append(LogEntry{Message: "INFO from container", Sequence: 1})

	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(Model)
	if got.dialog != dialogLogDetail {
		t.Fatalf("dialog = %v, want log detail", got.dialog)
	}
	if len(got.logModal.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(got.logModal.Entries))
	}
}

func TestLogModalOpensFromCentralLogsFocused(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.view = viewLogs
	m.width = 80
	m.height = 30
	m.centralLogs = NewLogViewerState(centralLogBufferMax, nil)
	ts := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	m.centralLogs.Append(LogEntry{ContainerName: "api", Message: "INFO hello", Timestamp: ts, Sequence: 1})

	model, _ := m.updateCentralLogs(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(Model)
	if got.dialog != dialogLogDetail {
		t.Fatalf("dialog = %v, want log detail", got.dialog)
	}
	if len(got.logModal.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(got.logModal.Entries))
	}
	if got.logModal.Entries[0].Message != "INFO hello" {
		t.Fatalf("unexpected entry %q", got.logModal.Entries[0].Message)
	}
}

func TestLogModalOpensWithMultiSelectionInOrder(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.view = viewLogs
	m.width = 80
	m.height = 30
	m.centralLogs = NewLogViewerState(centralLogBufferMax, nil)
	m.centralLogs.Append(
		LogEntry{Message: "one", Sequence: 1},
		LogEntry{Message: "two", Sequence: 2},
		LogEntry{Message: "three", Sequence: 3},
	)
	m.centralLogs.Selection = map[int64]bool{1: true, 3: true}

	model, _ := m.updateCentralLogs(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(Model)
	if len(got.logModal.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(got.logModal.Entries))
	}
	if got.logModal.Entries[0].Message != "one" || got.logModal.Entries[1].Message != "three" {
		t.Fatalf("order wrong: %q, %q", got.logModal.Entries[0].Message, got.logModal.Entries[1].Message)
	}
}

func TestLogModalEscCloses(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.height = 30
	m.dialog = dialogLogDetail
	m.logModal = LogModalState{Entries: []LogEntry{{Message: "x", Sequence: 1}}}

	model, _ := m.handleDialog(tea.KeyMsg{Type: tea.KeyEsc})
	got := model.(Model)
	if got.dialog != dialogNone {
		t.Fatal("esc must close modal")
	}
}

func TestLogModalJSONPrettyPrint(t *testing.T) {
	lines := formatLogModalBodyLines(LogEntry{Message: `{"ok":true,"n":1}`}, 60)
	body := strings.Join(lines, "\n")
	if !strings.Contains(body, `"ok"`) || !strings.Contains(body, "true") {
		t.Fatalf("expected pretty json, got %q", body)
	}
	if !strings.Contains(body, "\n") {
		t.Fatal("expected multiline json")
	}
}

func TestLogModalLogfmtSplit(t *testing.T) {
	msg := "a=1 b=2 c=3"
	lines := formatLogModalBodyLines(LogEntry{Message: msg}, 80)
	if len(lines) < 3 {
		t.Fatalf("expected split logfmt lines, got %d: %v", len(lines), lines)
	}
}

func TestLogModalWrapWidth40(t *testing.T) {
	msg := strings.Repeat("word ", 30)
	lines := formatLogModalBodyLines(LogEntry{Message: strings.TrimSpace(msg)}, 40)
	for _, line := range lines {
		if displayWidth(stripANSI(line)) > 40 {
			t.Fatalf("line wider than 40: %q (%d)", stripANSI(line), displayWidth(stripANSI(line)))
		}
	}
}

func TestLogModalTinyTerminalNoPanic(t *testing.T) {
	cases := [][2]int{{20, 8}, {1, 1}}
	for _, wh := range cases {
		m := modelWithLogModal(wh[0], wh[1], []LogEntry{logModalTestEntry()}, true)
		_ = m.View()
		_ = m.renderLogDetailDialog()
	}
}

func TestLogModalSnapshotUnaffectedByAppend(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.view = viewLogs
	m.width = 80
	m.height = 30
	m.centralLogs = NewLogViewerState(centralLogBufferMax, nil)
	m.centralLogs.Append(LogEntry{Message: "first", Sequence: 1})
	model, _ := m.updateCentralLogs(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(Model)
	got.centralLogs.Append(LogEntry{Message: "second", Sequence: 2})
	if len(got.logModal.Entries) != 1 {
		t.Fatalf("snapshot len = %d", len(got.logModal.Entries))
	}
	if got.logModal.Entries[0].Message != "first" {
		t.Fatalf("snapshot message = %q", got.logModal.Entries[0].Message)
	}
}

func assertLogModalOverlayInvariants(t *testing.T, m Model, entry LogEntry) {
	raw := m.renderDialogOverlay()
	lines := strings.Split(raw, "\n")
	if len(lines) != m.height {
		t.Fatalf("height %d: got %d lines, want %d", m.width, len(lines), m.height)
	}
	out := stripANSI(raw)
	var topLeft, bottomLeft int
	for i, line := range lines {
		if w := lipgloss.Width(line); w > m.width {
			t.Fatalf("line %d wider than terminal (%d > %d): %q", i+1, w, m.width, stripANSI(line))
		}
		plain := stripANSI(line)
		if strings.Contains(plain, "││") {
			t.Fatalf("line %d has doubled border: %q", i+1, plain)
		}
		topLeft += strings.Count(plain, "╭")
		bottomLeft += strings.Count(plain, "╰")
	}
	if topLeft != 1 {
		t.Fatalf("expected exactly one ╭, got %d", topLeft)
	}
	if bottomLeft != 1 {
		t.Fatalf("expected exactly one bottom-left corner, got %d", bottomLeft)
	}
	if !strings.Contains(out, "Log Detail") {
		t.Fatal("missing Log Detail title")
	}
	if !strings.Contains(out, entry.Timestamp.Format("15:04:05")) {
		t.Fatalf("missing time stamp in output")
	}
	msgPlain := entry.Message
	if strings.Contains(msgPlain, "{") {
		if !strings.Contains(out, "msg") && !strings.Contains(out, "level") && !strings.Contains(out, "{") {
			t.Fatalf("expected json body fragment in output: %q", out)
		}
	} else if !strings.Contains(out, msgPlain) {
		t.Fatalf("missing message text %q", msgPlain)
	}
}

func TestLogModalOverlayLayoutMatrix(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	entry := logModalTestEntry()
	widths := []int{150, 100, 80, 60, 40}
	heights := []int{40, 24, 12}
	for _, w := range widths {
		for _, h := range heights {
			name := fmt.Sprintf("%dx%d", w, h)
			t.Run(name, func(t *testing.T) {
				m := modelWithLogModal(w, h, []LogEntry{entry}, true)
				assertLogModalOverlayInvariants(t, m, entry)
			})
		}
	}
}

func TestLogModalLeftPaneLevelWhole(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 18, 55, 51, 0, time.UTC)
	entry := LogEntry{Message: "INFO hello", Timestamp: ts, Sequence: 1, ContainerName: "api"}
	for _, w := range []int{80, 100, 150} {
		m := modelWithLogModal(w, 40, []LogEntry{entry}, true)
		out := stripANSI(m.renderDialogOverlay())
		if !strings.Contains(out, "18:55:51 INF") {
			t.Fatalf("width %d: missing whole time+level, got fragment in output", w)
		}
	}
}

func TestLogModalTitleWording(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	one := modelWithLogModal(100, 40, []LogEntry{{Message: "a", Timestamp: ts, Sequence: 1}}, true)
	outOne := stripANSI(one.renderDialogOverlay())
	if !strings.Contains(outOne, "Log Detail · 1 line") {
		t.Fatalf("single entry title wrong: %q", outOne)
	}
	two := modelWithLogModal(100, 40, []LogEntry{
		{Message: "a", Timestamp: ts, Sequence: 1},
		{Message: "b", Timestamp: ts, Sequence: 2},
	}, true)
	outTwo := stripANSI(two.renderDialogOverlay())
	if !strings.Contains(outTwo, "Log Detail · 2 selected") {
		t.Fatalf("multi entry title wrong: %q", outTwo)
	}
	if !strings.Contains(outTwo, "1/2") {
		t.Fatalf("expected cursor position 1/2 in left pane: %q", outTwo)
	}
}

func logModalRenderedDialogLines(m Model) int {
	return len(strings.Split(m.renderLogDetailDialog(), "\n"))
}

func TestLogModalFixedHeightSameForShortAndLongBody(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	var longLines []string
	for i := 0; i < 200; i++ {
		longLines = append(longLines, fmt.Sprintf("line %d", i))
	}
	longMsg := strings.Join(longLines, "\n")
	widths := []int{150, 100, 80}
	heights := []int{40, 24}
	for _, w := range widths {
		for _, h := range heights {
			short := modelWithLogModal(w, h, []LogEntry{{Message: "one line body", Timestamp: ts, Sequence: 1}}, false)
			tall := modelWithLogModal(w, h, []LogEntry{{Message: longMsg, Timestamp: ts, Sequence: 1}}, false)
			_, shortRows := short.logModalInnerSize()
			_, tallRows := tall.logModalInnerSize()
			if shortRows != tallRows {
				t.Fatalf("%dx%d: list rows short=%d tall=%d", w, h, shortRows, tallRows)
			}
			shortH := logModalRenderedDialogLines(short)
			tallH := logModalRenderedDialogLines(tall)
			if shortH != tallH {
				t.Fatalf("%dx%d: dialog lines short=%d tall=%d", w, h, shortH, tallH)
			}
		}
	}
}

func TestLogModalBodyScrollClampsAndResetsOnItemChange(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	var body []string
	for i := 0; i < 80; i++ {
		body = append(body, fmt.Sprintf("line %d", i))
	}
	entries := []LogEntry{
		{Message: strings.Join(body, "\n"), Timestamp: ts, Sequence: 1},
		{Message: "second entry", Timestamp: ts.Add(time.Second), Sequence: 2},
	}
	m := modelWithLogModal(100, 40, entries, true)
	m.logModal.BodyScroll = 1_000
	m.logModal.clampBodyScroll(m.logModalMaxBodyScroll())
	maxScroll := m.logModalMaxBodyScroll()
	if m.logModal.BodyScroll != maxScroll {
		t.Fatalf("BodyScroll = %d, want clamped %d", m.logModal.BodyScroll, maxScroll)
	}
	m.logModal.BodyScroll = 5
	m.logModal.moveCursor(1)
	if m.logModal.BodyScroll != 0 {
		t.Fatalf("BodyScroll after item change = %d, want 0", m.logModal.BodyScroll)
	}
	m.logModal.Cursor = 0
	m.logModal.BodyScroll = 0
	nm, _ := m.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyPgDown})
	got := nm.(Model)
	if got.logModal.BodyScroll <= 0 {
		t.Fatalf("pgdown should increase scroll on long body, got %d", got.logModal.BodyScroll)
	}
	got.logModal.BodyScroll = 1_000
	got.logModal.clampBodyScroll(got.logModalMaxBodyScroll())
	if got.logModal.BodyScroll != got.logModalMaxBodyScroll() {
		t.Fatalf("scroll after pgdown should stay clamped, got %d", got.logModal.BodyScroll)
	}
}

func TestLogModalOverlayLinesWithinTerminalWidth(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	msg := strings.Repeat("word ", 40) + "\n" + strings.Repeat("x", 120)
	entry := LogEntry{Message: strings.TrimSpace(msg), Timestamp: ts, Sequence: 1, ContainerName: "api"}
	for _, w := range []int{150, 100, 80} {
		m := modelWithLogModal(w, 40, []LogEntry{entry}, true)
		raw := m.renderDialogOverlay()
		for i, line := range strings.Split(raw, "\n") {
			if displayWidth(stripANSI(line)) > w {
				t.Fatalf("width %d line %d: stripped width %d > terminal", w, i+1, displayWidth(stripANSI(line)))
			}
		}
	}
}

func TestLogModalMultiEntryLeftPane(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Message: "alpha", Timestamp: ts, Sequence: 1},
		{Message: "beta", Timestamp: ts.Add(time.Second), Sequence: 2},
		{Message: "gamma", Timestamp: ts.Add(2 * time.Second), Sequence: 3},
	}
	m := modelWithLogModal(100, 40, entries, true)
	m.logModal.Cursor = 1
	out := stripANSI(m.renderDialogOverlay())
	for _, tsStr := range []string{"10:00:00", "10:00:01", "10:00:02"} {
		if !strings.Contains(out, tsStr) {
			t.Fatalf("missing list row time %q", tsStr)
		}
	}
	cursorLine := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "10:00:01") && strings.Contains(line, glyphMarkerCursor) {
			cursorLine = line
			break
		}
	}
	if cursorLine == "" {
		t.Fatal("expected cursor highlight on second list row")
	}
}

func TestLogModalPaneFocusKeys(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Message: "a", Timestamp: ts, Sequence: 1, ContainerName: "one"},
		{Message: "b", Timestamp: ts.Add(time.Second), Sequence: 2, ContainerName: "two"},
	}
	m := modelWithLogModal(100, 40, entries, true)
	if m.logModal.Focus != logModalFocusLeft {
		t.Fatalf("open focus = %v, want left", m.logModal.Focus)
	}
	nm, _ := m.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyTab})
	got := nm.(Model)
	if got.logModal.Focus != logModalFocusRight {
		t.Fatalf("tab focus = %v, want right", got.logModal.Focus)
	}
	nm, _ = got.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyShiftTab})
	got = nm.(Model)
	if got.logModal.Focus != logModalFocusLeft {
		t.Fatalf("shift+tab focus = %v, want left", got.logModal.Focus)
	}
	narrow := modelWithLogModal(40, 40, entries, true)
	before := narrow.logModal.Cursor
	nm, _ = narrow.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyTab})
	got = nm.(Model)
	if got.logModal.Focus != logModalFocusLeft {
		t.Fatalf("single-pane tab must not change focus")
	}
	if got.logModal.Cursor == before {
		t.Fatal("single-pane tab should advance item")
	}
}

func TestLogModalRightPaneScrollKeys(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	var body []string
	for i := 0; i < 80; i++ {
		body = append(body, fmt.Sprintf("line %d", i))
	}
	entries := []LogEntry{
		{Message: strings.Join(body, "\n"), Timestamp: ts, Sequence: 1},
		{Message: "second", Timestamp: ts.Add(time.Second), Sequence: 2},
	}
	m := modelWithLogModal(100, 40, entries, false)
	m.logModal.Focus = logModalFocusRight
	m.logModal.Cursor = 0
	nm, _ := m.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := nm.(Model)
	if got.logModal.BodyScroll != 1 {
		t.Fatalf("j scroll = %d, want 1", got.logModal.BodyScroll)
	}
	if got.logModal.Cursor != 0 {
		t.Fatalf("cursor moved on right focus: %d", got.logModal.Cursor)
	}
	nm, _ = got.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeySpace})
	got = nm.(Model)
	if got.logModal.BodyScroll <= 1 {
		t.Fatalf("space should page body, scroll=%d", got.logModal.BodyScroll)
	}
	got.logModal.Focus = logModalFocusRight
	nm, _ = got.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	got = nm.(Model)
	if got.logModal.BodyScroll != got.logModalMaxBodyScroll() {
		t.Fatalf("G scroll = %d, want max %d", got.logModal.BodyScroll, got.logModalMaxBodyScroll())
	}
}

func TestLogModalLeftPaneItemResetsScroll(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	long := strings.Repeat("x\n", 40)
	entries := []LogEntry{
		{Message: long, Timestamp: ts, Sequence: 1},
		{Message: "short", Timestamp: ts.Add(time.Second), Sequence: 2},
	}
	m := modelWithLogModal(100, 40, entries, true)
	m.logModal.BodyScroll = 5
	nm, _ := m.handleLogDetailDialog(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := nm.(Model)
	if got.logModal.BodyScroll != 0 {
		t.Fatalf("item change should reset scroll, got %d", got.logModal.BodyScroll)
	}
	if got.logModal.Cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.logModal.Cursor)
	}
}

func TestLogModalPaneWidthsSumInner(t *testing.T) {
	for _, innerW := range []int{60, 80, 100, 150} {
		left, right, show := logModalPaneWidths(innerW, innerW)
		if !show {
			continue
		}
		if left+1+right != innerW {
			t.Fatalf("inner %d: left(%d)+1+right(%d) != inner", innerW, left, right)
		}
	}
}

func TestLogModalLeftRowUsesAvailableTagWidth(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 16, 22, 24, 0, time.UTC)
	longName := "pomodoist-selfhost-postgres-db"
	entry := LogEntry{
		ContainerName: longName,
		Message:       "INFO hello",
		Timestamp:     ts,
		Sequence:      1,
	}
	for _, w := range []int{80, 100, 150} {
		m := modelWithLogModal(w, 40, []LogEntry{entry}, true)
		innerW, _ := m.logModalInnerSize()
		leftW, _, show := logModalPaneWidths(innerW, w)
		if !show {
			t.Fatalf("width %d: expected split layout", w)
		}
		row := stripANSI(m.renderLogModalListRow(entry, leftW, true, 0, true))
		if w == 150 {
			if !strings.Contains(row, "pomodoist-self") {
				t.Fatalf("width %d: tag should use wide left pane: %q", w, row)
			}
		}
		if displayWidth(row) > leftW {
			t.Fatalf("width %d: row wider than pane %d > %d", w, displayWidth(row), leftW)
		}
	}
}

func TestLogModalMiddleTruncationKeepsSuffix(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{ContainerName: "pomodoist-selfhost-api", Message: "a", Timestamp: ts, Sequence: 1},
		{ContainerName: "pomodoist-selfhost-db", Message: "b", Timestamp: ts.Add(time.Second), Sequence: 2},
	}
	m := modelWithLogModal(100, 40, entries, true)
	innerW, _ := m.logModalInnerSize()
	leftW, _, _ := logModalPaneWidths(innerW, 100)
	out := stripANSI(m.renderLogModalListRow(entries[1], leftW, false, 1, true))
	if !strings.Contains(out, "-db") {
		t.Fatalf("expected distinguishing suffix in row: %q", out)
	}
}

func TestLogModalFooterFitsWidth80(t *testing.T) {
	applyTheme(config.FindTheme("dark-green"))
	ts := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	m := modelWithLogModal(80, 30, []LogEntry{{Message: "x", Timestamp: ts, Sequence: 1}}, true)
	innerW, _ := m.logModalInnerSize()
	footer := m.renderLogModalFooter()
	if displayWidth(stripANSI(footer)) > innerW {
		t.Fatalf("footer width %d > inner %d: %q", displayWidth(stripANSI(footer)), innerW, stripANSI(footer))
	}
	m.logModal.Focus = logModalFocusRight
	footer = m.renderLogModalFooter()
	if displayWidth(stripANSI(footer)) > innerW {
		t.Fatalf("right footer width %d > inner %d", displayWidth(stripANSI(footer)), innerW)
	}
}

func TestGoldenLogDetailDialog(t *testing.T) {
	withFixedNow(t, fixedNow)
	applyTheme(config.FindTheme("dark-green"))
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.view = viewLogs
	m.width = 80
	m.height = 30
	m.centralLogs = NewLogViewerState(centralLogBufferMax, nil)
	m.centralLogs.Append(LogEntry{
		ContainerName: "api",
		Message:       "INFO request complete",
		Timestamp:     fixedNow,
		Sequence:      1,
	})
	nm, ok := m.openLogDetailModal(&m.centralLogs, true)
	if !ok {
		t.Fatal("failed to open modal")
	}
	got := nm.View()
	if lipgloss.Width(got) == 0 {
		t.Fatal("empty render")
	}
	if !strings.Contains(got, "Log Detail") {
		t.Fatal("title missing")
	}
	path := goldenPath("log_detail_dialog_w80")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run: go test ./ui/ -run TestGoldenLogDetailDialog -update): %v", path, err)
	}
	if got != string(want) {
		t.Errorf("golden mismatch\n--- want ---\n%s\n--- got ---\n%s", visualize(string(want)), visualize(got))
	}
}
