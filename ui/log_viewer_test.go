package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSelectCentralLogTargetsUsesSelectedContainers(t *testing.T) {
	containers := []docker.ContainerInfo{
		{ID: "aaa111", Name: "api", State: "running"},
		{ID: "bbb222", Name: "worker", State: "exited"},
		{ID: "ccc333", Name: "db", State: "running"},
	}
	targets := SelectCentralLogTargets(containers, map[string]bool{"bbb222": true, "ccc333": true}, nil)
	if len(targets) != 2 {
		t.Fatalf("expected 2 selected targets, got %d", len(targets))
	}
	if targets[0].ID != "bbb222" || targets[1].ID != "ccc333" {
		t.Fatalf("unexpected target order: %#v", targets)
	}
}

func TestSelectCentralLogTargetsFallsBackToRunningContainers(t *testing.T) {
	containers := []docker.ContainerInfo{
		{ID: "aaa111", Name: "api", State: "running"},
		{ID: "bbb222", Name: "worker", State: "exited"},
		{ID: "ccc333", Name: "db", State: "running"},
	}
	targets := SelectCentralLogTargets(containers, nil, nil)
	if len(targets) != 2 {
		t.Fatalf("expected 2 running targets, got %d", len(targets))
	}
	if targets[0].ID != "aaa111" || targets[1].ID != "ccc333" {
		t.Fatalf("unexpected running targets: %#v", targets)
	}
}

func TestSelectCentralLogTargetsFallsBackWhenSelectionIsStale(t *testing.T) {
	containers := []docker.ContainerInfo{
		{ID: "aaa111", Name: "api", State: "running"},
		{ID: "bbb222", Name: "worker", State: "exited"},
		{ID: "ccc333", Name: "db", State: "running"},
	}
	targets := SelectCentralLogTargets(containers, map[string]bool{"missing": true}, nil)
	if len(targets) != 2 {
		t.Fatalf("expected fallback to 2 running targets, got %d", len(targets))
	}
	if targets[0].ID != "aaa111" || targets[1].ID != "ccc333" {
		t.Fatalf("unexpected fallback targets: %#v", targets)
	}
}

func TestSelectCentralLogTargetsFallsBackWhenSelectionValuesAreFalse(t *testing.T) {
	containers := []docker.ContainerInfo{
		{ID: "aaa111", Name: "api", State: "running"},
		{ID: "bbb222", Name: "worker", State: "exited"},
		{ID: "ccc333", Name: "db", State: "running"},
	}
	targets := SelectCentralLogTargets(containers, map[string]bool{"aaa111": false, "ccc333": false}, nil)
	if len(targets) != 2 {
		t.Fatalf("expected fallback to 2 running targets, got %d", len(targets))
	}
	if targets[0].ID != "aaa111" || targets[1].ID != "ccc333" {
		t.Fatalf("unexpected fallback targets: %#v", targets)
	}
}

func TestResolveLogTargetColorPrecedence(t *testing.T) {
	colors := map[string]string{
		"abc123": "#111111",
		"api":    "#222222",
		"abc":    "#333333",
		"bad":    "green",
	}
	target := LogTarget{ID: "abc123", Name: "api"}
	if got := ResolveLogTargetColor(colors, target); got != "#111111" {
		t.Fatalf("exact ID color = %q, want #111111", got)
	}
	target.ID = "abc999"
	if got := ResolveLogTargetColor(colors, target); got != "#222222" {
		t.Fatalf("exact name color = %q, want #222222", got)
	}
	target.Name = "sidecar"
	if got := ResolveLogTargetColor(colors, target); got != "#333333" {
		t.Fatalf("ID prefix color = %q, want #333333", got)
	}
	target.ID = "bad999"
	if got := ResolveLogTargetColor(colors, target); got == "green" || got == "" {
		t.Fatalf("invalid configured color was not ignored: %q", got)
	}
}

func TestLogViewerAppendTrimAndFollow(t *testing.T) {
	viewer := NewLogViewerState(3, []LogTarget{{ID: "api1", Name: "api"}})
	for i, msg := range []string{"one", "two", "three", "four"} {
		viewer.Append(LogEntry{ContainerID: "api1", ContainerName: "api", Message: msg, Sequence: int64(i + 1)})
	}
	if len(viewer.Entries) != 3 {
		t.Fatalf("expected trimmed buffer length 3, got %d", len(viewer.Entries))
	}
	if viewer.Entries[0].Message != "two" {
		t.Fatalf("expected oldest retained message two, got %q", viewer.Entries[0].Message)
	}
	start, end, total := viewer.VisibleWindow(2)
	if start != 1 || end != 3 || total != 3 {
		t.Fatalf("follow visible window = (%d,%d,%d), want (1,3,3)", start, end, total)
	}
}

func TestLogViewerScrollPauseAndResume(t *testing.T) {
	viewer := NewLogViewerState(10, nil)
	for i, msg := range []string{"one", "two", "three", "four", "five"} {
		viewer.Append(LogEntry{Message: msg, Sequence: int64(i + 1)})
	}
	viewer.ScrollBy(-1, 3)
	if viewer.Follow {
		t.Fatalf("scrolling up should pause follow")
	}
	if viewer.Scroll != 1 {
		t.Fatalf("scroll after one row up = %d, want 1", viewer.Scroll)
	}
	viewer.ScrollBy(1, 3)
	if !viewer.Follow {
		t.Fatalf("scrolling to bottom should resume follow")
	}
	viewer.ScrollHome(3)
	if viewer.Follow || viewer.Scroll != 0 {
		t.Fatalf("home should pause at top, got scroll=%d follow=%v", viewer.Scroll, viewer.Follow)
	}
	viewer.ScrollEnd()
	if !viewer.Follow {
		t.Fatalf("end should resume follow")
	}
}

func TestLogViewerScrollHomePausesWhenAllRowsFit(t *testing.T) {
	viewer := NewLogViewerState(10, nil)
	viewer.Append(
		LogEntry{Message: "one", Sequence: 1},
		LogEntry{Message: "two", Sequence: 2},
	)
	viewer.ScrollHome(5)
	if viewer.Follow {
		t.Fatalf("home should pause follow even when all rows fit")
	}
	if viewer.Scroll != 0 {
		t.Fatalf("home should stay at top, got scroll=%d", viewer.Scroll)
	}
}

func TestLogViewerScrollUpPausesWhenAllRowsFit(t *testing.T) {
	viewer := NewLogViewerState(10, nil)
	viewer.Append(
		LogEntry{Message: "one", Sequence: 1},
		LogEntry{Message: "two", Sequence: 2},
	)
	viewer.ScrollBy(-1, 5)
	if viewer.Follow {
		t.Fatalf("scrolling up should pause follow even when all rows fit")
	}
	if viewer.Scroll != 0 {
		t.Fatalf("scrolling up without overflow should remain at top, got scroll=%d", viewer.Scroll)
	}
}

func TestLogViewerFilterMatchesNameAndMessage(t *testing.T) {
	viewer := NewLogViewerState(10, nil)
	viewer.Append(
		LogEntry{ContainerName: "api", Message: "GET /health", Sequence: 1},
		LogEntry{ContainerName: "worker", Message: "job failed", Sequence: 2},
	)
	viewer.SetFilter("api")
	if got := viewer.FilteredEntries(); len(got) != 1 || got[0].ContainerName != "api" {
		t.Fatalf("filter by container name returned %#v", got)
	}
	viewer.SetFilter("failed")
	if got := viewer.FilteredEntries(); len(got) != 1 || got[0].ContainerName != "worker" {
		t.Fatalf("filter by message returned %#v", got)
	}
}

func TestLogViewerVisibleEntriesUsesFilterAndScroll(t *testing.T) {
	viewer := NewLogViewerState(10, nil)
	viewer.Append(
		LogEntry{Message: "alpha", Sequence: 1},
		LogEntry{Message: "beta", Sequence: 2},
		LogEntry{Message: "gamma", Sequence: 3},
		LogEntry{Message: "delta", Sequence: 4},
	)
	viewer.SetFilter("a")
	viewer.ScrollHome(2)
	got := viewer.VisibleEntries(2)
	if len(got) != 2 || got[0].Message != "alpha" || got[1].Message != "beta" {
		t.Fatalf("visible filtered rows = %#v", got)
	}
	viewer.ScrollEnd()
	got = viewer.VisibleEntries(2)
	if len(got) != 2 || got[0].Message != "gamma" || got[1].Message != "delta" {
		t.Fatalf("visible follow rows = %#v", got)
	}
}

func TestSortLogEntriesUsesTimestampThenSequence(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	entries := []LogEntry{
		{Message: "timestamp seq 20", Timestamp: now, Sequence: 20},
		{Message: "untimestamped seq 30", Sequence: 30},
		{Message: "later timestamp", Timestamp: now.Add(time.Second), Sequence: 1},
		{Message: "timestamp seq 10", Timestamp: now, Sequence: 10},
		{Message: "untimestamped seq 10", Sequence: 10},
	}
	SortLogEntries(entries)
	want := []string{
		"timestamp seq 10",
		"timestamp seq 20",
		"later timestamp",
		"untimestamped seq 10",
		"untimestamped seq 30",
	}
	for i, wantMessage := range want {
		if entries[i].Message != wantMessage {
			t.Fatalf("entry %d = %q, want %q; order: %#v", i, entries[i].Message, wantMessage, entries)
		}
	}
}

func TestDetailLogsScrollMovesOneRenderedRow(t *testing.T) {
	m := Model{
		height:    10,
		detailTab: tabLogs,
		logViewer: NewLogViewerState(20, nil),
	}
	for i := 1; i <= 12; i++ {
		m.logViewer.Append(LogEntry{Message: string(rune('0' + i)), Sequence: int64(i)})
	}

	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyHome})
	moved, _ := model.(Model).updateDetail(tea.KeyMsg{Type: tea.KeyDown})
	gotModel := moved.(Model)
	rows := gotModel.logViewer.VisibleEntries(gotModel.detailLogContentRows())

	if gotModel.logViewer.Follow {
		t.Fatalf("expected one-step down from home to remain paused")
	}
	if len(rows) == 0 || rows[0].Message != "2" {
		t.Fatalf("expected first rendered row to move to message 2, got %#v", rows)
	}
}

func TestDetailLogsMouseScrollMovesOneRenderedRow(t *testing.T) {
	m := Model{
		view:      viewDetail,
		height:    10,
		detailTab: tabLogs,
		logViewer: NewLogViewerState(20, nil),
	}
	for i := 1; i <= 12; i++ {
		m.logViewer.Append(LogEntry{Message: string(rune('0' + i)), Sequence: int64(i)})
	}

	m.logViewer.ScrollHome(m.detailLogContentRows())
	model, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	gotModel := model.(Model)
	rows := gotModel.logViewer.VisibleEntries(gotModel.detailLogContentRows())

	if gotModel.logViewer.Follow {
		t.Fatalf("expected one mouse-wheel step down from home to remain paused")
	}
	if len(rows) == 0 || rows[0].Message != "2" {
		t.Fatalf("expected first rendered row to move to message 2, got %#v", rows)
	}
}

func TestUpdateListLKeyOpensCentralLogs(t *testing.T) {
	m := Model{
		cfg: &config.Config{
			ContainerColors: map[string]string{},
		},
		containers: []docker.ContainerInfo{
			{ID: "aaa111", Name: "api", State: "running"},
			{ID: "bbb222", Name: "worker", State: "exited"},
		},
		selected: map[string]bool{
			"bbb222": true,
		},
	}

	model, cmd := m.updateList(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	got := model.(Model)

	if got.view != viewLogs {
		t.Fatalf("view = %v, want viewLogs", got.view)
	}
	if len(got.centralLogTargets) != 1 || got.centralLogTargets[0].ID != "bbb222" {
		t.Fatalf("central targets = %#v, want selected container", got.centralLogTargets)
	}
	if cmd == nil {
		t.Fatal("expected centralized logs commands to be scheduled")
	}
}

func TestOpenCentralLogsNoTargetsShowsSystemMessage(t *testing.T) {
	m := Model{
		cfg: &config.Config{
			ContainerColors: map[string]string{},
		},
		containers: []docker.ContainerInfo{
			{ID: "aaa111", Name: "api", State: "exited"},
		},
	}

	model, cmd := m.openCentralLogs()
	got := model.(Model)
	if got.view != viewLogs {
		t.Fatalf("view = %v, want viewLogs", got.view)
	}
	if cmd != nil {
		t.Fatal("expected no command when there are no selected or running targets")
	}
	if len(got.centralLogs.Entries) != 1 || !got.centralLogs.Entries[0].System {
		t.Fatalf("expected one system entry, got %#v", got.centralLogs.Entries)
	}
}

func TestCentralLogsFilterLifecycle(t *testing.T) {
	m := Model{
		view: viewLogs,
		centralLogs: NewLogViewerState(20, []LogTarget{
			{ID: "aaa111", Name: "api"},
		}),
	}
	m.centralLogs.Append(
		LogEntry{ContainerID: "aaa111", ContainerName: "api", Message: "GET /health", Sequence: 1},
		LogEntry{ContainerID: "aaa111", ContainerName: "api", Message: "job failed", Sequence: 2},
	)

	model, _ := m.updateCentralLogs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	filtering := model.(Model)
	if !filtering.centralLogFiltering {
		t.Fatal("expected / to enable central log filtering")
	}

	model, _ = filtering.updateCentralLogs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	typed := model.(Model)
	if typed.centralLogFilter != "f" || typed.centralLogs.Filter != "f" {
		t.Fatalf("filter text = %q, viewer filter = %q", typed.centralLogFilter, typed.centralLogs.Filter)
	}
	if got := typed.centralLogs.FilteredEntries(); len(got) != 1 || got[0].Message != "job failed" {
		t.Fatalf("filtered entries = %#v, want one failed row", got)
	}

	model, _ = typed.updateCentralLogs(tea.KeyMsg{Type: tea.KeyBackspace})
	cleared := model.(Model)
	if cleared.centralLogFilter != "" || cleared.centralLogs.Filter != "" {
		t.Fatalf("filter should clear after backspace, got %q / %q", cleared.centralLogFilter, cleared.centralLogs.Filter)
	}

	model, _ = cleared.updateCentralLogs(tea.KeyMsg{Type: tea.KeyEsc})
	done := model.(Model)
	if done.centralLogFiltering {
		t.Fatal("expected esc to exit central log filtering")
	}
}

func TestHiddenContainerFilter(t *testing.T) {
	viewer := NewLogViewerState(20, []LogTarget{
		{ID: "a1", Name: "api"},
		{ID: "b2", Name: "worker"},
	})
	viewer.Append(
		LogEntry{ContainerID: "a1", ContainerName: "api", Message: "one", Sequence: 1},
		LogEntry{ContainerID: "b2", ContainerName: "worker", Message: "two", Sequence: 2},
	)
	viewer.ToggleContainer("a1")
	if got := viewer.FilteredEntries(); len(got) != 1 || got[0].Message != "two" {
		t.Fatalf("filtered = %#v, want worker line only", got)
	}
	viewer.ShowAllContainers()
	if got := viewer.FilteredEntries(); len(got) != 2 {
		t.Fatalf("expected all containers visible, got %d", len(got))
	}
}

func TestFormatLogLinesForCopyJoinsMultipleLines(t *testing.T) {
	entries := []LogEntry{
		{ContainerName: "api", Message: "one", Sequence: 1},
		{ContainerName: "api", Message: "two", Sequence: 2},
	}
	got := formatLogLinesForCopy(entries)
	if !strings.Contains(got, "one") || !strings.Contains(got, "two") || !strings.Contains(got, "\n") {
		t.Fatalf("copy text = %q", got)
	}
}

func TestCentralLogsToggleContainerByNumber(t *testing.T) {
	m := Model{
		view:   viewLogs,
		width:  100,
		height: 24,
		centralLogTargets: []LogTarget{
			{ID: "a1", Name: "api"},
			{ID: "b2", Name: "worker"},
		},
		centralLogs: NewLogViewerState(20, []LogTarget{
			{ID: "a1", Name: "api"},
			{ID: "b2", Name: "worker"},
		}),
	}
	m.centralLogs.Append(
		LogEntry{ContainerID: "a1", Message: "one", Sequence: 1},
		LogEntry{ContainerID: "b2", Message: "two", Sequence: 2},
	)

	model, _ := m.updateCentralLogs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	got := model.(Model)
	if !got.centralLogs.IsContainerHidden("a1") {
		t.Fatal("expected container 1 hidden")
	}
	if entries := got.centralLogs.FilteredEntries(); len(entries) != 1 || entries[0].Message != "two" {
		t.Fatalf("filtered entries = %#v", entries)
	}
}

func TestContainerLegendLinesWraps(t *testing.T) {
	targets := make([]LogTarget, 5)
	for i := range targets {
		targets[i] = LogTarget{ID: fmt.Sprintf("id%d", i), Name: fmt.Sprintf("service-%d", i), Color: "#00E676"}
	}
	if lines := containerLegendLines(targets, 40); lines < 2 {
		t.Fatalf("expected wrapped legend lines >= 2 at width 40, got %d", lines)
	}
}

func TestCentralLogsMouseScrollMovesFocus(t *testing.T) {
	m := Model{
		view:        viewLogs,
		height:      24,
		width:       100,
		centralLogs: NewLogViewerState(20, nil),
	}
	for i := 1; i <= 12; i++ {
		m.centralLogs.Append(LogEntry{Message: string(rune('0' + i)), Sequence: int64(i)})
	}

	m.centralLogs.ScrollHome(m.centralLogContentRows())
	model, _ := m.handleMouse(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	got := model.(Model)

	if got.centralLogs.Follow {
		t.Fatalf("expected wheel-down from home to pause follow")
	}
	if got.centralLogs.Focused != 1 {
		t.Fatalf("expected focus on line 2 (index 1), got %d", got.centralLogs.Focused)
	}
}

func TestBuildLogCellsColoredTag(t *testing.T) {
	targets := map[string]LogTarget{
		"abc": {ID: "abc", Name: "api", Color: "#FF5252"},
	}
	cells := buildLogCells(LogEntry{
		ContainerID:   "abc",
		ContainerName: "api",
		Message:       "hello",
	}, 80, true, targets, nil)
	if len(cells) < 2 {
		t.Fatalf("expected tag+message cells, got %d", len(cells))
	}
	foundTag := false
	for _, c := range cells {
		if c.BG == lipgloss.Color("#FF5252") {
			foundTag = true
		}
	}
	if !foundTag {
		t.Fatalf("expected colored tag cell, got %#v", cells)
	}
}
