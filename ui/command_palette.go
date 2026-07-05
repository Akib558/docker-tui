package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) getCommands() []Command {
	commands := []Command{
		{Name: "refresh", Description: "Refresh container list", Run: func(m Model) (Model, tea.Cmd) {
			m.loading = true
			return m, m.refreshContainers()
		}},
		{Name: "images", Description: "Open images view", Run: func(m Model) (Model, tea.Cmd) {
			m.view = viewImages
			m.imgCursor = 0
			m.loading = true
			return m, m.fetchImages()
		}},
		{Name: "volumes", Description: "Open volumes view", Run: func(m Model) (Model, tea.Cmd) {
			m.view = viewVolumes
			m.volCursor = 0
			m.loading = true
			return m, m.fetchVolumes()
		}},
		{Name: "networks", Description: "Open networks view", Run: func(m Model) (Model, tea.Cmd) {
			m.view = viewNetworks
			m.netCursor = 0
			m.loading = true
			return m, m.fetchNetworks()
		}},
		{Name: "logs", Description: "Open centralized logs", Run: func(m Model) (Model, tea.Cmd) {
			model, cmd := m.openCentralLogs()
			return model.(Model), cmd
		}},
		{Name: "events", Description: "Open docker events stream", Run: func(m Model) (Model, tea.Cmd) {
			m.view = viewEvents
			m.eventsCursor = 0
			return m, m.startEventStream()
		}},
		{Name: "notifications", Description: "Open notification center", Run: func(m Model) (Model, tea.Cmd) {
			m.view = viewNotifications
			m.notifyCursor = len(m.notifyHistory) - 1
			if m.notifyCursor < 0 {
				m.notifyCursor = 0
			}
			return m, nil
		}},
		{Name: "theme", Description: "Open theme picker", Run: func(m Model) (Model, tea.Cmd) {
			m.dialog = dialogTheme
			return m, nil
		}},
		{Name: "help", Description: "Show keyboard reference", Run: func(m Model) (Model, tea.Cmd) {
			m.dialog = dialogHelp
			return m, nil
		}},
		{Name: "system-prune", Description: "Run docker system prune", Run: func(m Model) (Model, tea.Cmd) {
			m.dialog = dialogConfirm
			m.confirmMsg = "Run docker system prune?\n\nRemoves stopped containers, dangling images, unused networks, and orphaned volumes."
			m.confirmOK = m.pruneSystem()
			return m, nil
		}},
	}

	if c := m.selectedContainer(); c != nil {
		commands = append(commands, []Command{
			{Name: "start", Description: "Start selected container", Run: func(m Model) (Model, tea.Cmd) {
				if c.State != "running" {
					return m, m.startContainer(c.ID, c.Name)
				}
				return m, nil
			}},
			{Name: "stop", Description: "Stop selected container", Run: func(m Model) (Model, tea.Cmd) {
				if c.State == "running" {
					return m, m.stopContainer(c.ID, c.Name)
				}
				return m, nil
			}},
			{Name: "restart", Description: "Restart selected container", Run: func(m Model) (Model, tea.Cmd) {
				return m, m.restartContainer(c.ID, c.Name)
			}},
			{Name: "pause", Description: "Pause/unpause selected container", Run: func(m Model) (Model, tea.Cmd) {
				model, cmd := m.doPauseUnpause()
				return model.(Model), cmd
			}},
			{Name: "kill", Description: "Kill selected container", Run: func(m Model) (Model, tea.Cmd) {
				return m, m.killContainer(c.ID, c.Name)
			}},
			{Name: "remove", Description: "Remove selected container", Run: func(m Model) (Model, tea.Cmd) {
				model, cmd := m.confirmRemove()
				return model.(Model), cmd
			}},
			{Name: "exec", Description: "Exec into selected container", Run: func(m Model) (Model, tea.Cmd) {
				model, cmd := m.execIntoContainer()
				return model.(Model), cmd
			}},
			{Name: "details", Description: "Open container details", Run: func(m Model) (Model, tea.Cmd) {
				model, cmd := m.openDetail(*c)
				return model.(Model), cmd
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
