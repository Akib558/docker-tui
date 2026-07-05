package ui

import (
	"testing"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
)

func TestThemesApplyWithoutPanic(t *testing.T) {
	for _, theme := range config.Themes {
		applyTheme(&theme)
		if colorPrimary == "" || colorBarTrack == "" {
			t.Fatalf("theme %q: colors not applied", theme.Name)
		}
	}
}

func TestFilteredCacheInvalidatesOnSort(t *testing.T) {
	m := NewModel(&config.Config{Theme: "dark-green"})
	m.containers = []docker.ContainerInfo{
		{ID: "a", Name: "alpha", State: "running"},
		{ID: "b", Name: "beta", State: "running"},
	}
	m.stats = make(map[string]*docker.ContainerResourceStats)
	m.rebuildFilteredCache()
	first := m.filteredContainers()
	m.sortMode = sortCPU
	m.rebuildFilteredCache()
	second := m.filteredContainers()
	if len(first) != len(second) {
		t.Fatalf("cache length mismatch")
	}
}
