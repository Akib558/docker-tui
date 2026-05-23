package main

import (
	"fmt"
	"os"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.Load()
	p := tea.NewProgram(ui.NewModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
