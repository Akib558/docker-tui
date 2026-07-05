package ui

import (
	"strings"
	"testing"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

func TestRenderMatrixListView(t *testing.T) {
	widths := []int{40, 60, 80, 120, 200}
	heights := []int{12, 24, 40}
	for _, w := range widths {
		for _, h := range heights {
			m := NewModel(&config.Config{Theme: "dark-green"})
			m.width = w
			m.height = h
			m.loading = false
			for i := 0; i < 8; i++ {
				m.containers = append(m.containers, docker.ContainerInfo{
					ID:     string(rune('a' + i)),
					Name:   "svc-" + string(rune('a'+i)),
					Image:  "img:latest",
					State:  "running",
					Status: "Up 1h",
				})
			}
			m.stats = map[string]*docker.ContainerResourceStats{
				"a": {CPUPercent: 12, MemPercent: 34},
			}
			out := m.viewList()
			lines := strings.Split(out, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				if got := lipgloss.Width(line); got > w {
					t.Fatalf("%dx%d: line width %d > %d: %q", w, h, got, w, line)
				}
			}
			if !strings.Contains(out, "quit") {
				t.Fatalf("%dx%d: help bar missing", w, h)
			}
		}
	}
}

func TestTableHeaderAlignsWithRow(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 120
	m.loading = false
	m.containers = []docker.ContainerInfo{{
		ID: "abc", Name: "web", Image: "nginx", State: "running", Status: "Up",
	}}
	cols := m.calcColumns()
	header := m.renderTableHeader(cols)
	row := m.renderTableRow(m.containers[0], false, cols, 0)
	if lipgloss.Width(row) != m.width {
		t.Fatalf("row width = %d, want %d", lipgloss.Width(row), m.width)
	}
	if lipgloss.Width(header) > m.width {
		t.Fatalf("header width %d exceeds terminal %d", lipgloss.Width(header), m.width)
	}
}
