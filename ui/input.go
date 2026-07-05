package ui

import (
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

func appendTextInput(dst *string, msg tea.KeyMsg) bool {
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		*dst += string(msg.Runes)
		return true
	}
	return false
}

func backspaceTextInput(dst *string) {
	if len(*dst) == 0 {
		return
	}
	_, size := utf8.DecodeLastRuneInString(*dst)
	*dst = (*dst)[:len(*dst)-size]
}
