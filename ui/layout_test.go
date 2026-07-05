package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

func TestRowWAndTableW(t *testing.T) {
	if got := RowW(80); got != 76 {
		t.Fatalf("RowW(80) = %d, want 76", got)
	}
	if got := TableW(80); got != 74 {
		t.Fatalf("TableW(80) = %d, want 74", got)
	}
}

func TestRenderSelectableRowFitsTerminal(t *testing.T) {
	content := "nginx"
	for _, w := range []int{40, 80, 120} {
		for _, kind := range []ListRowKind{ListRowNormal, ListRowCursor, ListRowSelected, ListRowCursorSelected} {
			row := renderSelectableRow(w, kind, content)
			got := lipgloss.Width(row)
			if got > w {
				t.Fatalf("row width %d exceeds terminal %d (kind=%d)", got, w, kind)
			}
			if kind == ListRowCursor && got != w {
				t.Fatalf("cursor row width %d != terminal %d", got, w)
			}
		}
	}
}

func TestCursorRowSpansFullWidth(t *testing.T) {
	w := 80
	row := renderSelectableRow(w, ListRowCursor, "nginx")
	if got := lipgloss.Width(row); got != w {
		t.Fatalf("cursor row width = %d, want full width %d", got, w)
	}
}

func TestRenderSelectableRowFitsDialogBox(t *testing.T) {
	dialogW := 44
	row := renderSelectableRow(dialogW, ListRowCursor, "dark-green")
	if got := lipgloss.Width(row); got > dialogW {
		t.Fatalf("dialog row width %d exceeds dialog %d", got, dialogW)
	}
}

func TestBodyRowsMinimumOne(t *testing.T) {
	if got := bodyRows(10, 20); got != 1 {
		t.Fatalf("bodyRows should floor at 1, got %d", got)
	}
}

func TestClampIndex(t *testing.T) {
	if got := clampIndex(5, 3); got != 2 {
		t.Fatalf("clampIndex(5,3) = %d, want 2", got)
	}
}

func TestOnResizeClampsCursors(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.loading = false
	m.containers = make([]docker.ContainerInfo, 5)
	for i := range m.containers {
		m.containers[i] = docker.ContainerInfo{ID: string(rune('a' + i)), Name: "c", State: "running"}
	}
	m.cursor = 99
	m.onResize()
	if m.cursor != 4 {
		t.Fatalf("cursor = %d after resize, want 4", m.cursor)
	}
}

func TestHelpBarVisibleOnSmallTerminal(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.height = 20
	m.loading = false
	for i := 0; i < 15; i++ {
		m.containers = append(m.containers, docker.ContainerInfo{
			ID: string(rune('a' + i)), Name: "svc", Image: "img", State: "running", Status: "Up",
		})
	}
	m.cursor = 5
	out := m.View()
	if !strings.Contains(out, "quit") {
		t.Fatalf("help bar missing on small terminal")
	}
}

func TestListFrameReservesScrollChrome(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.height = 24
	m.loading = false
	for i := 0; i < 30; i++ {
		m.containers = append(m.containers, docker.ContainerInfo{
			ID: string(rune('a' + i)), Name: "c", State: "running",
		})
	}
	frame := m.listFrame(30)
	if frame.BodyRows >= m.height {
		t.Fatalf("body rows %d should be less than height %d", frame.BodyRows, m.height)
	}
}

func TestNetworksViewRenders(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.height = 24
	m.loading = false
	m.networks = []docker.NetworkResource{{ID: "1", Name: "bridge", Driver: "bridge", Scope: "local"}}
	out := m.viewNetworks()
	if !strings.Contains(out, "bridge") {
		t.Fatalf("expected network name in output")
	}
}

func TestDetailLogsTabFitsViewport(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 100
	m.height = 24
	m.inspected = &docker.ContainerInfo{ID: "abc", Name: "web", State: "running"}
	m.detailTab = tabLogs
	m.logViewer = NewLogViewerState(100, nil)
	for i := 0; i < 50; i++ {
		m.logViewer.Append(LogEntry{
			Message:  fmt.Sprintf("log line %d with some content", i),
			Sequence: int64(i + 1),
		})
	}

	content := m.renderLogsTab(max(m.width-10, 24))
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	avail := m.detailBoxInnerHeight()
	if len(lines) > avail {
		t.Fatalf("logs tab rendered %d lines, want <= %d", len(lines), avail)
	}
}

func TestDetailBoxInnerHeightReasonable(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.height = 30
	m.inspected = &docker.ContainerInfo{ID: "abc", Name: "web", State: "running"}
	got := m.detailBoxInnerHeight()
	if got < 5 || got > m.height {
		t.Fatalf("detailBoxInnerHeight = %d, height = %d", got, m.height)
	}
}
