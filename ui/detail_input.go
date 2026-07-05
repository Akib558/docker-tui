package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) detailAllowsContainerActions() bool {
	switch m.detailTab {
	case tabInfo, tabResources, tabEnv:
		return true
	default:
		return false
	}
}

func isTerminalPrintableKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes:
		return len(msg.Runes) > 0
	case tea.KeySpace, tea.KeyTab:
		return true
	default:
		return false
	}
}

func appendTerminalInput(dst *string, msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes:
		if len(msg.Runes) > 0 {
			*dst += string(msg.Runes)
			return true
		}
	case tea.KeySpace:
		*dst += " "
		return true
	case tea.KeyTab:
		*dst += "\t"
		return true
	}
	return false
}

func (m Model) handleDetailTerminalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.terminalInputFocused = false
		return m, nil
	case "ctrl+\\":
		m.stopTerminalSession()
		m.terminalInputFocused = false
		m.notify("Terminal detached", false)
		return m, nil
	case "enter":
		if m.terminalActive {
			line := m.terminalInput
			m.terminalInput = ""
			return m, m.sendTerminalInput(line + "\n")
		}
		return m, nil
	case "backspace":
		backspaceTextInput(&m.terminalInput)
		return m, nil
	default:
		if appendTerminalInput(&m.terminalInput, msg) {
			return m, nil
		}
	}
	return m, nil
}
