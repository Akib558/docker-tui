package ui

import (
	"sort"
	"strings"

	"github.com/akib558/docker-tui/docker"
)

func (m *Model) invalidateFilteredCache() {
	m.filteredCache = nil
}

func (m *Model) rebuildFilteredCache() {
	m.filteredCache = m.computeFilteredContainers()
	m.filteredCacheKey = filteredCacheKey{
		filter:   m.filterText,
		sortMode: m.sortMode,
		n:        len(m.containers),
	}
}

func (m *Model) computeFilteredContainers() []docker.ContainerInfo {
	var result []docker.ContainerInfo
	if m.filterText == "" {
		result = make([]docker.ContainerInfo, len(m.containers))
		copy(result, m.containers)
	} else {
		q := strings.ToLower(m.filterText)
		for _, c := range m.containers {
			if strings.Contains(strings.ToLower(c.Name), q) ||
				strings.Contains(strings.ToLower(c.Image), q) ||
				strings.Contains(strings.ToLower(c.State), q) {
				result = append(result, c)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		switch m.sortMode {
		case sortName:
			pi, pj := statePriority(result[i].State), statePriority(result[j].State)
			if pi != pj {
				return pi < pj
			}
			return result[i].Name < result[j].Name
		case sortState:
			pi, pj := statePriority(result[i].State), statePriority(result[j].State)
			if pi != pj {
				return pi < pj
			}
			return result[i].Name < result[j].Name
		case sortCPU:
			cpuI := 0.0
			cpuJ := 0.0
			if s := m.stats[result[i].ID]; s != nil {
				cpuI = s.CPUPercent
			}
			if s := m.stats[result[j].ID]; s != nil {
				cpuJ = s.CPUPercent
			}
			if cpuI == cpuJ {
				return result[i].Name < result[j].Name
			}
			return cpuI > cpuJ
		case sortMemory:
			memI := 0.0
			memJ := 0.0
			if s := m.stats[result[i].ID]; s != nil {
				memI = s.MemPercent
			}
			if s := m.stats[result[j].ID]; s != nil {
				memJ = s.MemPercent
			}
			if memI == memJ {
				return result[i].Name < result[j].Name
			}
			return memI > memJ
		case sortImage:
			if result[i].Image == result[j].Image {
				return result[i].Name < result[j].Name
			}
			return result[i].Image < result[j].Image
		default:
			return result[i].Name < result[j].Name
		}
	})
	return result
}

func (m *Model) invalidateDashboardCache() {
	m.dashboardCache = ""
	m.dashboardCacheW = 0
}

func (m *Model) cachedDashboard(w int) string {
	if m.dashboardCache != "" && m.dashboardCacheW == w {
		return m.dashboardCache
	}
	m.dashboardCache = m.renderDashboard(w)
	m.dashboardCacheW = w
	return m.dashboardCache
}

func (m *Model) pruneHistoryKeys() {
	ids := make(map[string]bool, len(m.containers))
	for _, c := range m.containers {
		ids[c.ID] = true
	}
	for id := range m.cpuHistory {
		if !ids[id] {
			delete(m.cpuHistory, id)
		}
	}
	for id := range m.memHistory {
		if !ids[id] {
			delete(m.memHistory, id)
		}
	}
}

func (m Model) needsStats() bool {
	switch m.view {
	case viewList, viewDetail:
		return true
	default:
		return false
	}
}
