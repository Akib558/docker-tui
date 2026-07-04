package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) getCommands() []Command {
	commands := []Command{
		{Name: "refresh", Description: "Refresh container list", Action: func() tea.Cmd {
			m.loading = true
			return m.refreshContainers()
		}},
		{Name: "images", Description: "Open images view", Action: func() tea.Cmd {
			m.view = viewImages
			m.imgCursor = 0
			m.loading = true
			return m.fetchImages()
		}},
		{Name: "volumes", Description: "Open volumes view", Action: func() tea.Cmd {
			m.view = viewVolumes
			m.volCursor = 0
			m.loading = true
			return m.fetchVolumes()
		}},
		{Name: "networks", Description: "Open networks view", Action: func() tea.Cmd {
			m.view = viewNetworks
			m.netCursor = 0
			m.loading = true
			return m.fetchNetworks()
		}},
		{Name: "logs", Description: "Open centralized logs", Action: func() tea.Cmd {
			model, cmd := m.openCentralLogs()
			m = model.(Model)
			return cmd
		}},
		{Name: "notifications", Description: "Open notification center", Action: func() tea.Cmd {
			m.view = viewNotifications
			m.notifyCursor = len(m.notifyHistory) - 1
			return nil
		}},
		{Name: "theme", Description: "Open theme picker", Action: func() tea.Cmd {
			m.dialog = dialogTheme
			return nil
		}},
		{Name: "help", Description: "Show keyboard reference", Action: func() tea.Cmd {
			m.dialog = dialogHelp
			return nil
		}},
		{Name: "system-prune", Description: "Run docker system prune", Action: func() tea.Cmd {
			m.dialog = dialogConfirm
			m.confirmMsg = "Run docker system prune?\n\nRemoves stopped containers, dangling images, unused networks, and orphaned volumes."
			m.confirmOK = m.pruneSystem()
			return nil
		}},
	}

	// Add container-specific commands if a container is selected
	if c := m.selectedContainer(); c != nil {
		commands = append(commands, []Command{
			{Name: "start", Description: "Start selected container", Action: func() tea.Cmd {
				if c.State != "running" {
					return m.startContainer(c.ID, c.Name)
				}
				return nil
			}},
			{Name: "stop", Description: "Stop selected container", Action: func() tea.Cmd {
				if c.State == "running" {
					return m.stopContainer(c.ID, c.Name)
				}
				return nil
			}},
			{Name: "restart", Description: "Restart selected container", Action: func() tea.Cmd {
				return m.restartContainer(c.ID, c.Name)
			}},
			{Name: "pause", Description: "Pause/unpause selected container", Action: func() tea.Cmd {
				model, cmd := m.doPauseUnpause()
				m = model.(Model)
				return cmd
			}},
			{Name: "kill", Description: "Kill selected container", Action: func() tea.Cmd {
				return m.killContainer(c.ID, c.Name)
			}},
			{Name: "remove", Description: "Remove selected container", Action: func() tea.Cmd {
				model, cmd := m.confirmRemove()
				m = model.(Model)
				return cmd
			}},
			{Name: "exec", Description: "Exec into selected container", Action: func() tea.Cmd {
				model, cmd := m.execIntoContainer()
				m = model.(Model)
				return cmd
			}},
			{Name: "details", Description: "Open container details", Action: func() tea.Cmd {
				model, cmd := m.openDetail(*c)
				m = model.(Model)
				return cmd
			}},
		}...)
	}

	return commands
}

func (m Model) filterCommands(query string) []Command {
	if query == "" {
		return m.getCommands()
	}

	allCommands := m.getCommands()
	query = strings.ToLower(query)
	var filtered []Command

	for _, cmd := range allCommands {
		if strings.Contains(strings.ToLower(cmd.Name), query) ||
			strings.Contains(strings.ToLower(cmd.Description), query) {
			filtered = append(filtered, cmd)
		}
	}

	return filtered
}
