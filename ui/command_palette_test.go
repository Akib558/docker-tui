package ui

import (
	"testing"

	"github.com/akib558/docker-tui/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandPaletteImagesSwitchesView(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.dialog = dialogCommandPalette
	m.commandPaletteResults = m.getCommands()
	m.commandPaletteCursor = 1 // "images" after "refresh"

	model, _ := m.handleDialog(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(Model)

	if updated.view != viewImages {
		t.Fatalf("view = %v, want viewImages", updated.view)
	}
	if !updated.loading {
		t.Fatalf("expected loading=true after palette images")
	}
}

func TestCommandPaletteThemeOpensDialog(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.dialog = dialogCommandPalette
	results := m.getCommands()
	var themeIdx int
	for i, c := range results {
		if c.Name == "theme" {
			themeIdx = i
			break
		}
	}
	m.commandPaletteCursor = themeIdx
	m.commandPaletteResults = results

	model, _ := m.handleDialog(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(Model)

	if updated.dialog != dialogTheme {
		t.Fatalf("dialog = %v, want dialogTheme", updated.dialog)
	}
}
