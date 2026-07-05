package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

type histData struct {
	CPU map[string][]float64 `json:"cpu"`
	Mem map[string][]float64 `json:"mem"`
}

func histPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "docker-tui", "history.json")
}

func (m Model) loadHistory() tea.Cmd {
	return func() tea.Msg {
		data, err := os.ReadFile(histPath())
		if err != nil {
			return loadHistMsg{
				cpu: make(map[string][]float64),
				mem: make(map[string][]float64),
			}
		}
		var h histData
		if err := json.Unmarshal(data, &h); err != nil {
			return loadHistMsg{
				cpu: make(map[string][]float64),
				mem: make(map[string][]float64),
			}
		}
		return loadHistMsg{cpu: h.CPU, mem: h.Mem}
	}
}

func (m Model) saveHistory() tea.Cmd {
	cpu := copyHistMap(m.cpuHistory)
	mem := copyHistMap(m.memHistory)
	return func() tea.Msg {
		path := histPath()
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		h := histData{CPU: cpu, Mem: mem}
		data, _ := json.Marshal(h)
		_ = os.WriteFile(path, data, 0644)
		return nil
	}
}

func copyHistMap(src map[string][]float64) map[string][]float64 {
	if len(src) == 0 {
		return make(map[string][]float64)
	}
	dst := make(map[string][]float64, len(src))
	for k, v := range src {
		cp := make([]float64, len(v))
		copy(cp, v)
		dst[k] = cp
	}
	return dst
}
