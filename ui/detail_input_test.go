package ui

import (
	"io"
	"strings"
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
	if got.terminalInput != "" {
		t.Fatalf("unfocused s must not enter the shell, got %q", got.terminalInput)
	}
	if got.terminalInputFocused {
		t.Fatal("unfocused s must not steal focus")
	}
	if got.view != viewDetail {
		t.Fatal("expected to remain on detail view")
	}
}

func TestUnfocusedDigitSwitchesTab(t *testing.T) {
	m := Model{
		view:           viewDetail,
		detailTab:      tabTerminal,
		inspected:      &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
		terminalActive: true,
	}
	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	got := model.(Model)
	if got.detailTab != tabInfo {
		t.Fatalf("detailTab = %d, want tabInfo (%d)", got.detailTab, tabInfo)
	}
	if got.terminalInputFocused {
		t.Fatal("digit must not focus the terminal")
	}
	if got.terminalInput != "" {
		t.Fatalf("terminalInput = %q, want empty", got.terminalInput)
	}
}

func TestUnfocusedTabAdvances(t *testing.T) {
	m := Model{
		view:           viewDetail,
		detailTab:      tabTerminal,
		terminalActive: true,
		inspected:      &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyTab})
	got := model.(Model)
	if got.detailTab != tabDiff {
		t.Fatalf("detailTab = %d, want tabDiff (%d)", got.detailTab, tabDiff)
	}
	if got.terminalInputFocused {
		t.Fatal("tab must not focus the terminal")
	}
}

func TestFocusedDigitStaysInInput(t *testing.T) {
	m := Model{
		view:                 viewDetail,
		detailTab:            tabTerminal,
		terminalActive:       true,
		terminalInputFocused: true,
		inspected:            &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	got := model.(Model)
	if got.detailTab != tabTerminal {
		t.Fatalf("detailTab = %d, want tabTerminal", got.detailTab)
	}
	if got.terminalInput != "1" {
		t.Fatalf("terminalInput = %q, want 1", got.terminalInput)
	}
}

func TestCtrlBracketReleasesFocus(t *testing.T) {
	m := Model{
		view:                 viewDetail,
		detailTab:            tabTerminal,
		terminalActive:       true,
		terminalInputFocused: true,
	}
	model, _ := m.updateDetail(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	got := model.(Model)
	if got.terminalInputFocused {
		t.Fatal("ctrl+] must release focus")
	}
	if got.view != viewDetail || got.detailTab != tabTerminal {
		t.Fatal("ctrl+] must stay on the terminal tab")
	}
}

func TestOnTabSwitchDoesNotStartOrFocus(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabTerminal,
		inspected: &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	model, cmd := m.onTabSwitch()
	got := model.(Model)
	if cmd != nil {
		t.Fatal("landing on terminal tab must not start a shell")
	}
	if got.terminalInputFocused {
		t.Fatal("landing on terminal tab must not focus input")
	}
}

func TestOnTabSwitchDoesNotKillSession(t *testing.T) {
	stopped := false
	m := Model{
		view:           viewDetail,
		detailTab:      tabInfo,
		terminalActive: true,
		terminalCancel: func() { stopped = true },
	}
	model, _ := m.onTabSwitch()
	got := model.(Model)
	if stopped {
		t.Fatal("leaving the terminal tab must not detach the session")
	}
	if !got.terminalActive {
		t.Fatal("session must stay marked active after leaving the tab")
	}
}

func TestIAttachKeepsFocusAfterStart(t *testing.T) {
	m := Model{
		view:                 viewDetail,
		detailTab:            tabTerminal,
		terminalInputFocused: true,
		width:                80,
		height:               24,
	}
	model, _ := m.Update(terminalStartMsg{
		cancel: func() {},
		writer: io.Discard,
		shell:  "/bin/sh",
	})
	got := model.(Model)
	if !got.terminalInputFocused {
		t.Fatal("i attach focus must survive session start")
	}
}

func TestXKeepsFocusFalseAfterStart(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabTerminal,
		width:     80,
		height:    24,
	}
	model, _ := m.Update(terminalStartMsg{
		cancel: func() {},
		writer: io.Discard,
		shell:  "/bin/sh",
	})
	got := model.(Model)
	if got.terminalInputFocused {
		t.Fatal("x connect must not gain focus on session start")
	}
}

func TestTerminalStartMsgDoesNotFocus(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabTerminal,
		width:     80,
		height:    24,
	}
	model, _ := m.Update(terminalStartMsg{
		cancel: func() {},
		writer: io.Discard,
		shell:  "/bin/sh",
	})
	got := model.(Model)
	if !got.terminalActive {
		t.Fatal("session should be active")
	}
	if got.terminalInputFocused {
		t.Fatal("connect must not steal focus")
	}
}

func TestIAttachesAndFocuses(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabTerminal,
		inspected: &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	model, cmd := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	got := model.(Model)
	if !got.terminalInputFocused {
		t.Fatal("i must focus")
	}
	if cmd == nil {
		t.Fatal("i must start the session when inactive")
	}
}

func TestXAttachesWithoutFocus(t *testing.T) {
	m := Model{
		view:      viewDetail,
		detailTab: tabTerminal,
		inspected: &docker.ContainerInfo{ID: "abc", Name: "api", State: "running"},
	}
	model, cmd := m.updateDetail(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	got := model.(Model)
	if got.terminalInputFocused {
		t.Fatal("x must not focus")
	}
	if cmd == nil {
		t.Fatal("x must start the session when inactive")
	}
}

func TestIdleStatusLine(t *testing.T) {
	m := Model{
		detailTab: tabTerminal,
		width:     80,
		height:    24,
	}
	out := m.renderTerminalTab(&docker.ContainerInfo{State: "running"}, 60)
	if !strings.Contains(out, "i attach · x connect") {
		t.Fatalf("idle status missing, got %q", out)
	}
	if strings.Contains(out, "start typing") {
		t.Fatal("old auto-focus hint must go")
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
