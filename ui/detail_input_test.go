package ui

import (
	"testing"

	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTerminalTabDoesNotStopContainerOnS(t *testing.T) {
	m := Model{
		view:           viewDetail,
		detailTab:      tabTerminal,
		inspected:      &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
		terminalActive: true,
	}
	model, cmd := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := model.(Model)
	if cmd != nil {
		t.Fatal("expected no stop/start command when typing s in terminal tab")
	}
	if got.terminalInput != "s" {
		t.Fatalf("terminal input = %q, want s", got.terminalInput)
	}
}

func TestTerminalInputAcceptsSpace(t *testing.T) {
	m := Model{
		view:                 viewDetail,
		detailTab:            tabTerminal,
		terminalActive:       true,
		terminalInputFocused: true,
	}
	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeySpace})
	got := model.(Model)
	if got.terminalInput != " " {
		t.Fatalf("terminal input = %q, want space", got.terminalInput)
	}
}

func TestTerminalBackspaceEditsBufferNotNavigate(t *testing.T) {
	m := Model{
		view:                 viewDetail,
		detailTab:            tabTerminal,
		terminalActive:       true,
		terminalInputFocused: true,
		terminalInput:        "ls",
	}
	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyBackspace})
	got := model.(Model)
	if got.view != viewDetail {
		t.Fatalf("view = %v, want detail", got.view)
	}
	if got.terminalInput != "l" {
		t.Fatalf("terminal input = %q, want l", got.terminalInput)
	}
}

func TestContainerActionsBlockedOnLogsTab(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabLogs,
		inspected: &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	model, cmd := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd != nil {
		t.Fatal("expected container stop to be blocked on logs tab")
	}
	if model.(Model).view != viewDetail {
		t.Fatal("expected to remain on detail view")
	}
}

func TestContainerActionsAllowedOnInfoTab(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabInfo,
		inspected: &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	_, cmd := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected stop command on info tab")
	}
}
