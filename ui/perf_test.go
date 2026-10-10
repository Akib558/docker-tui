package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

func recomputeVisibleNaive(s *LogViewerState) []LogEntry {
	out := make([]LogEntry, 0, len(s.Entries))
	for _, entry := range s.Entries {
		if s.HiddenContainers != nil && s.HiddenContainers[entry.ContainerID] {
			continue
		}
		if s.Filter == "" {
			out = append(out, entry)
			continue
		}
		if s.UseRegex && s.filterRE != nil {
			if s.filterRE.MatchString(entry.Message) ||
				s.filterRE.MatchString(entry.ContainerName) ||
				s.filterRE.MatchString(entry.ContainerID) {
				out = append(out, entry)
			}
			continue
		}
		q := strings.ToLower(s.Filter)
		if strings.Contains(entry.nameLower, q) ||
			strings.Contains(entry.idLower, q) ||
			strings.Contains(entry.msgLower, q) {
			out = append(out, entry)
		}
	}
	return out
}

func TestVisibleCacheMatchesNaive(t *testing.T) {
	targets := []LogTarget{
		{ID: "c1", Name: "api"},
		{ID: "c2", Name: "worker"},
		{ID: "c3", Name: "db"},
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		filter string
		regex  bool
		hidden map[string]bool
	}{
		{"", false, nil},
		{"error", false, nil},
		{"API", false, nil},
		{"c1", false, map[string]bool{"c2": true}},
		{".*warn.*", true, nil},
	}
	for _, tc := range cases {
		v := NewLogViewerState(500, targets)
		for i := 0; i < 200; i++ {
			id := targets[i%len(targets)].ID
			name := targets[i%len(targets)].Name
			msg := "info line"
			if i%7 == 0 {
				msg = "error spike"
			}
			if i%11 == 0 {
				msg = "warn slow"
			}
			v.Append(LogEntry{
				ContainerID: id, ContainerName: name,
				Timestamp: base.Add(time.Duration(i) * time.Second),
				Message:   msg, Sequence: int64(i + 1),
			})
		}
		if tc.hidden != nil {
			v.HiddenContainers = tc.hidden
			v.invalidateVisible()
		}
		v.SetFilter(tc.filter)
		v.SetUseRegex(tc.regex)
		got := v.FilteredEntries()
		want := recomputeVisibleNaive(&v)
		if len(got) != len(want) {
			t.Fatalf("filter=%q regex=%v len got %d want %d", tc.filter, tc.regex, len(got), len(want))
		}
		for i := range got {
			if got[i].Sequence != want[i].Sequence {
				t.Fatalf("filter=%q mismatch at %d: got seq %d want %d", tc.filter, i, got[i].Sequence, want[i].Sequence)
			}
		}
	}
}

func TestLogBatchPreservesOrderAndRearms(t *testing.T) {
	m := Model{
		view:        viewDetail,
		detailTab:   tabLogs,
		liveLogging: true,
		logViewer:   NewLogViewerState(detailLogBufferMax, nil),
	}
	var next tea.Cmd
	msg := logBatchMsg{
		entries: []LogEntry{
			{Message: "a", Sequence: 1},
			{Message: "b", Sequence: 2},
		},
		next: func() tea.Msg { return logStreamDoneMsg{} },
	}
	next = msg.next
	msg.next = next
	model, cmd := m.Update(msg)
	got := model.(Model)
	if len(got.logViewer.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got.logViewer.Entries))
	}
	if got.logViewer.Entries[0].Message != "a" || got.logViewer.Entries[1].Message != "b" {
		t.Fatalf("order wrong: %#v", got.logViewer.Entries)
	}
	if cmd == nil {
		t.Fatal("expected re-arm command")
	}
}

func TestStaleStatsGenerationDropped(t *testing.T) {
	m := Model{statsGen: 2, stats: make(map[string]*docker.ContainerResourceStats)}
	model, _ := m.Update(statsMsg{
		gen:   1,
		stats: map[string]*docker.ContainerResourceStats{"x": {CPUPercent: 99}},
		done:  true,
	})
	got := model.(Model)
	if got.stats["x"] != nil {
		t.Fatal("stale stats batch should be ignored")
	}
}
