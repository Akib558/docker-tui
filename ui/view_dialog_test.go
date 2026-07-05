package ui

import (
	"strings"
	"testing"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

func TestHelpDialogFitsViewport(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.height = 24
	m.dialog = dialogHelp

	out := m.renderDialogOverlay()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) > m.height {
		t.Fatalf("help overlay %d lines exceeds height %d", len(lines), m.height)
	}
	if !strings.Contains(out, "Keyboard Reference") {
		t.Fatalf("help title missing")
	}
	if !strings.Contains(out, "j/k") {
		t.Fatalf("scroll hint missing")
	}
}

func TestHelpDialogScrolls(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 80
	m.height = 20
	m.dialog = dialogHelp

	first := m.renderHelpDialog()
	m.helpScroll = m.helpDialogMaxScroll()
	second := m.renderHelpDialog()
	if first == second && m.helpDialogMaxScroll() > 0 {
		t.Fatalf("scrolled help should differ from top")
	}
}

func TestDetailTabsNoUnderline(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.width = 120
	m.inspected = &docker.ContainerInfo{Name: "web", State: "running"}
	m.detailTab = tabLogs

	out := m.renderDetailTabs(100)
	if strings.Contains(out, "▄") || strings.Contains(out, "▀") {
		t.Fatalf("tab row should not use block border chars: %q", out)
	}
	if !strings.Contains(out, "Logs") {
		t.Fatalf("active tab name missing")
	}
	if lipgloss.Width(out) > 100 {
		t.Fatalf("tab row width %d exceeds budget 100", lipgloss.Width(out))
	}
}
