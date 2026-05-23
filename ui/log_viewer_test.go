package ui

import (
	"testing"
	"time"

	"github.com/akib558/docker-tui/docker"
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
