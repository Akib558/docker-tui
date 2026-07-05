package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// version is injected at release time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	v := resolveVersion()
	if *showVersion {
		fmt.Println(v)
		return
	}

	ui.SetVersion(v)
	cfg := config.Load()
	p := tea.NewProgram(ui.NewModel(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolveVersion prefers the ldflags-injected version used by release builds,
// then falls back to the module version embedded by `go install ...@vX.Y.Z`.
func resolveVersion() string {
	if version != "dev" && version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
			return mv
		}
	}
	return version
}
