package ui

import (
	"fmt"

	"github.com/akib/docker-tui/docker"
)

func (m *Model) checkAlerts(id string, s *docker.ContainerResourceStats) {
	if s.CPUPercent >= m.cfg.AlertCPU && !m.alertShown[id+"_cpu"] {
		name := m.containerName(id)
		m.notify(fmt.Sprintf("HIGH CPU: %s %.0f%%", name, s.CPUPercent), true)
		m.alertShown[id+"_cpu"] = true
	} else if s.CPUPercent < m.cfg.AlertCPU*0.8 {
		delete(m.alertShown, id+"_cpu")
	}
	if s.MemPercent >= m.cfg.AlertMem && !m.alertShown[id+"_mem"] {
		name := m.containerName(id)
		m.notify(fmt.Sprintf("HIGH MEM: %s %.0f%%", name, s.MemPercent), true)
		m.alertShown[id+"_mem"] = true
	} else if s.MemPercent < m.cfg.AlertMem*0.8 {
		delete(m.alertShown, id+"_mem")
	}
}

func (m Model) containerName(id string) string {
	for _, c := range m.containers {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}
