// Package ui implements the Bubble Tea terminal UI for docker-tui.
package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Update ──────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.invalidateDashboardCache()
		m.onResize()
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case initMsg:
		m.client = msg.client
		m.containers = msg.containers
		m.overview = msg.overview
		m.systemMem = msg.sysMem
		m.systemLoad = msg.sysLoad
		m.loading = false
		m.lastRefresh = time.Now()
		m.fetchStats = true
		return m, m.collectStats()

	case loadHistMsg:
		m.cpuHistory = msg.cpu
		m.memHistory = msg.mem
		return m, nil

	case containersMsg:
		prevContainers := m.containers
		m.containers = []docker.ContainerInfo(msg)
		m.handleContainerStateTransitions(prevContainers, m.containers)
		m.loading = false
		m.lastRefresh = time.Now()
		m.containerNames = make(map[string]string, len(m.containers))
		for _, c := range m.containers {
			m.containerNames[c.ID] = c.Name
		}
		m.rebuildFilteredCache()
		m.pruneHistoryKeys()
		m.invalidateDashboardCache()
		m.clampCursorToFiltered()
		if m.width > 0 {
			m.dashboardCache = m.renderDashboard(m.width)
			m.dashboardCacheW = m.width
		}
		return m, nil

	case imagesMsg:
		m.images = []docker.ImageInfo(msg)
		m.loading = false
		return m, nil

	case volumesMsg:
		m.volumes = []docker.VolumeInfo(msg)
		m.loading = false
		return m, nil

	case networksMsg:
		m.networks = []docker.NetworkResource(msg)
		m.loading = false
		return m, nil

	case statsMsg:
		m.fetchStats = false
		m.stats = msg.stats
		m.systemMem = msg.sysMem
		m.systemLoad = msg.sysLoad
		for id, s := range msg.stats {
			m.cpuHistory[id] = appendHist(m.cpuHistory[id], s.CPUPercent)
			m.memHistory[id] = appendHist(m.memHistory[id], s.MemPercent)
			m.checkAlerts(id, s)
		}
		if m.sortMode == sortCPU || m.sortMode == sortMemory {
			m.rebuildFilteredCache()
		}
		m.invalidateDashboardCache()
		// save history every ~10 ticks to avoid disk thrash
		if m.tickCount%10 == 0 {
			return m, m.saveHistory()
		}
		return m, nil

	case inspectMsg:
		m.inspected = msg.info
		m.loading = false
		return m, nil

	case logsMsg:
		m.logViewer = NewLogViewerState(detailLogBufferMax, nil)
		m.logViewer.Append([]LogEntry(msg)...)
		return m, nil

	case logStreamStartMsg:
		m.logCancel = msg.cancel
		m.liveLogging = true
		return m, msg.next

	case logLineMsg:
		m.logViewer.Append(msg.entry)
		if m.view == viewDetail && m.detailTab == tabLogs && m.liveLogging {
			return m, msg.next
		}
		return m, nil

	case logStreamDoneMsg:
		m.liveLogging = false
		return m, nil

	case centralLogTailMsg:
		m.centralLogs.AppendSorted(msg.entries...)
		return m, nil

	case centralLogStreamStartMsg:
		m.centralLogCancels = append(m.centralLogCancels, msg.cancel)
		return m, msg.next

	case centralLogLineMsg:
		m.centralLogs.AppendSorted(msg.entry)
		if m.view == viewLogs {
			return m, msg.next
		}
		return m, nil

	case centralLogStreamDoneMsg:
		if msg.err != nil {
			m.centralLogs.Append(systemLogEntry(msg.target, fmt.Sprintf("stream ended: %v", msg.err)))
		}
		return m, nil

	case terminalStartMsg:
		if m.terminalCancel != nil {
			m.terminalCancel()
		}
		m.terminalCancel = msg.cancel
		m.terminalWriter = msg.writer
		m.terminalActive = true
		m.terminalFollow = true
		m.terminalShell = msg.shell
		m.terminalInputFocused = m.detailTab == tabTerminal
		if m.terminalOutput == "" {
			m.terminalOutput = fmt.Sprintf("Connected to shell: %s\n", msg.shell)
		}
		m.syncTerminalScroll()
		return m, msg.next

	case terminalChunkMsg:
		m.terminalOutput += msg.chunk
		if len(m.terminalOutput) > terminalBufferMax {
			m.terminalOutput = m.terminalOutput[len(m.terminalOutput)-terminalBufferMax:]
		}
		m.syncTerminalScroll()
		if m.view == viewDetail && m.detailTab == tabTerminal && m.terminalActive {
			return m, msg.next
		}
		return m, nil

	case terminalDoneMsg:
		m.stopTerminalSession()
		if msg.err != nil {
			m.notify(fmt.Sprintf("Terminal closed: %v", msg.err), true)
		}
		return m, nil

	case newEventMsg:
		m.events = append(m.events, msg.ev)
		if len(m.events) > 500 {
			m.events = m.events[1:]
		}
		return m, msg.next

	case eventStreamStartMsg:
		m.eventsCancel = msg.cancel
		return m, msg.next

	case diffMsg:
		m.diff = []docker.DiffEntry(msg)
		return m, nil

	case topMsg:
		m.processTop = msg.top
		m.processLoaded = true
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.reconnecting = true
		m.reconnectAttempts++
		m.notify(fmt.Sprintf("Connection lost. Reconnecting (attempt %d)...", m.reconnectAttempts), true)
		return m, m.reconnect()

	case reconnectMsg:
		if msg.success {
			if m.client != nil {
				m.client.Close()
			}
			m.client = msg.client
			m.reconnecting = false
			m.reconnectAttempts = 0
			m.err = nil
			m.notify("Reconnected to Docker", false)
			return m, m.refreshContainers()
		}
		m.notify(fmt.Sprintf("Reconnect failed: %v. Retrying...", msg.err), true)
		return m, m.reconnect()

	case actionDoneMsg:
		m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
		return m, m.refreshContainers()

	case imageActionDoneMsg:
		m.imagePullProgress = ""
		m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
		return m, m.fetchImages()

	case pullProgressMsg:
		m.imagePullProgress = msg.text
		if msg.next != nil {
			return m, msg.next
		}
		return m, nil

	case volumeActionDoneMsg:
		m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
		return m, m.fetchVolumes()

	case networkActionDoneMsg:
		m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
		return m, m.fetchNetworks()

	case execDoneMsg:
		if msg.err != nil {
			m.notify(fmt.Sprintf("Exec error: %v", msg.err), true)
		}
		return m, nil

	case tickMsg:
		m.tickCount++
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd(m.refreshInterval))
		if m.client != nil {
			if m.view == viewList || m.view == viewDetail || m.view == viewLogs {
				cmds = append(cmds, m.refreshContainers())
			}
			if !m.fetchStats && m.needsStats() {
				m.fetchStats = true
				cmds = append(cmds, m.collectStats())
			}
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if time.Since(m.notifyTime) > 4*time.Second {
			m.notification = ""
		}
		switch msg.String() {
		case "ctrl+c":
			return m.quit()
		case "?":
			if m.dialog == dialogNone && !m.filtering && !m.volFiltering && !m.netFiltering {
				m.dismissHint()
				m.dialog = dialogHelp
				m.helpScroll = 0
				return m, nil
			}
		case ":":
			if m.dialog == dialogNone && !m.filtering && !m.volFiltering && !m.netFiltering {
				m.dismissHint()
				m.dialog = dialogCommandPalette
				m.commandPaletteText = ""
				m.commandPaletteCursor = 0
				m.commandPaletteResults = m.getCommands()
				return m, nil
			}
		}
		if m.dialog != dialogNone {
			return m.handleDialog(msg)
		}
		if m.filtering && m.view == viewList {
			return m.handleFilter(msg)
		}
		if m.volFiltering && m.view == viewVolumes {
			return m.handleVolFilter(msg)
		}
		if m.netFiltering && m.view == viewNetworks {
			return m.handleNetFilter(msg)
		}
		switch m.view {
		case viewList:
			return m.updateList(msg)
		case viewDetail:
			return m.updateDetail(msg)
		case viewImages:
			return m.updateImages(msg)
		case viewEvents:
			return m.updateEvents(msg)
		case viewLogs:
			return m.updateCentralLogs(msg)
		case viewVolumes:
			return m.updateVolumes(msg)
		case viewNetworks:
			return m.updateNetworks(msg)
		case viewNotifications:
			return m.updateNotifications(msg)
		}
	}
	return m, nil
}

// ── Mouse ───────────────────────────────────────────────────────────────

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.view == viewLogs {
			m.centralLogs.MoveFocus(-1, m.centralLogContentRows())
		} else if m.view == viewDetail {
			if m.detailTab == tabLogs {
				m.logViewer.MoveFocus(-1, m.detailLogContentRows())
			} else if m.detailTab == tabTerminal {
				m.terminalFollow = false
				if m.detailScroll > 0 {
					m.detailScroll--
				}
				m.syncTerminalScroll()
			} else if m.detailScroll > 0 {
				m.detailScroll--
			}
		} else {
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case tea.MouseButtonWheelDown:
		if m.view == viewLogs {
			m.centralLogs.MoveFocus(1, m.centralLogContentRows())
		} else if m.view == viewDetail {
			if m.detailTab == tabLogs {
				m.logViewer.MoveFocus(1, m.detailLogContentRows())
			} else if m.detailTab == tabTerminal {
				if !m.terminalFollow {
					m.detailScroll++
				}
				m.syncTerminalScroll()
			} else {
				m.detailScroll++
			}
		} else {
			if m.cursor < len(m.containers)-1 {
				m.cursor++
			}
		}
	case tea.MouseButtonLeft:
		if m.view == viewList && m.dialog == dialogNone && !m.filtering {
			rowOffset := 9
			clickedRow := msg.Y - rowOffset
			if clickedRow >= 0 && clickedRow < len(m.containers) {
				m.cursor = clickedRow
			}
		}
	}
	return m, nil
}

// ── List ────────────────────────────────────────────────────────────────

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m.quit()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filteredContainers())-1 {
			m.cursor++
		}
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		if n := len(m.filteredContainers()); n > 0 {
			m.cursor = n - 1
		}
	case "enter", "l":
		if c := m.selectedContainer(); c != nil {
			return m.openDetail(*c)
		}
	case " ":
		if c := m.selectedContainer(); c != nil {
			if m.selected[c.ID] {
				delete(m.selected, c.ID)
			} else {
				m.selected[c.ID] = true
			}
		}
	case "a":
		if len(m.selected) > 0 {
			m.selected = make(map[string]bool)
		} else {
			for _, c := range m.containers {
				m.selected[c.ID] = true
			}
		}
	case "s":
		return m.toggleStartStop()
	case "R":
		return m.doRestart()
	case "P":
		return m.doPauseUnpause()
	case "K":
		return m.doKill()
	case "X":
		m.dialog = dialogConfirm
		m.confirmMsg = "Run docker system prune?\n\nRemoves stopped containers, dangling images, unused networks, and orphaned volumes."
		m.confirmOK = m.pruneSystem()
		return m, nil
	case "d":
		return m.confirmRemove()
	case "e":
		return m.execIntoContainer()
	case "r":
		m.loading = true
		return m, m.refreshContainers()
	case "i":
		m.view = viewImages
		m.imgCursor = 0
		m.loading = true
		return m, m.fetchImages()
	case "v":
		m.view = viewVolumes
		m.volCursor = 0
		m.loading = true
		return m, m.fetchVolumes()
	case "n":
		m.view = viewNetworks
		m.netCursor = 0
		m.loading = true
		return m, m.fetchNetworks()
	case "D":
		m.view = viewEvents
		m.eventsCursor = 0
		m.selected = make(map[string]bool)
		return m, m.startEventStream()
	case "L":
		return m.openCentralLogs()
	case "N":
		m.view = viewNotifications
		m.notifyCursor = len(m.notifyHistory) - 1
		return m, nil
	case "/":
		m.filtering = true
		m.filterText = ""
	case "C":
		m.filtering = false
		m.filterText = ""
		m.cursor = 0
		m.rebuildFilteredCache()
	case "c":
		m.groupByCompose = !m.groupByCompose
	case "S":
		m.sortMode = (m.sortMode + 1) % sortModeCount
		m.rebuildFilteredCache()
	case "t":
		m.dialog = dialogTheme
	case "+":
		m.cfg.StepRefresh(true)
		m.refreshInterval = m.cfg.RefreshDuration()
		m.notify("Refresh: "+config.FormatRefreshInterval(m.refreshInterval), false)
		go config.Save(m.cfg)
		m.clampCursorToFiltered()
		return m, tickCmd(m.refreshInterval)
	case "-":
		m.cfg.StepRefresh(false)
		m.refreshInterval = m.cfg.RefreshDuration()
		m.notify("Refresh: "+config.FormatRefreshInterval(m.refreshInterval), false)
		go config.Save(m.cfg)
		m.clampCursorToFiltered()
		return m, tickCmd(m.refreshInterval)
	}
	m.clampCursorToFiltered()
	return m, nil
}

func (m Model) openDetail(c docker.ContainerInfo) (tea.Model, tea.Cmd) {
	m.view = viewDetail
	m.detailScroll = 0
	m.detailTab = tabInfo
	m.logViewer = NewLogViewerState(detailLogBufferMax, []LogTarget{{
		ID:    c.ID,
		Name:  c.Name,
		State: c.State,
		Color: ResolveLogTargetColor(m.cfg.ContainerColors, LogTarget{ID: c.ID, Name: c.Name, State: c.State}),
	}})
	m.diff = nil
	m.processTop = docker.ContainerTop{}
	m.processLoaded = false
	m.terminalInput = ""
	m.terminalOutput = ""
	m.terminalShell = ""
	m.terminalFollow = true
	m.stopLogStreaming()
	m.stopTerminalSession()
	m.loading = true
	return m, tea.Batch(m.inspectContainer(c.ID), m.fetchLogs(c.ID))
}

func (m Model) openCentralLogs() (tea.Model, tea.Cmd) {
	targets := SelectCentralLogTargets(m.containers, m.selected, m.cfg.ContainerColors)
	m.stopCentralLogStreaming()
	m.centralLogTargets = targets
	m.centralLogs = NewLogViewerState(centralLogBufferMax, targets)
	m.centralLogFiltering = false
	m.centralLogFilter = ""
	m.centralLogRegex = false
	m.view = viewLogs
	if len(targets) == 0 {
		m.centralLogs.Append(LogEntry{
			Message: "No selected or running containers available.",
			System:  true,
		})
		return m, nil
	}
	return m, tea.Batch(m.fetchCentralLogTails(targets), m.startCentralLogStreams(targets))
}

func (m Model) leaveCentralLogs() (tea.Model, tea.Cmd) {
	m.stopCentralLogStreaming()
	m.view = viewList
	return m, nil
}

func (m Model) toggleStartStop() (tea.Model, tea.Cmd) {
	if len(m.selected) > 0 {
		var cmds []tea.Cmd
		for _, c := range m.containers {
			if m.selected[c.ID] {
				if c.State == "running" {
					cmds = append(cmds, m.stopContainer(c.ID, c.Name))
				} else {
					cmds = append(cmds, m.startContainer(c.ID, c.Name))
				}
			}
		}
		m.selected = make(map[string]bool)
		return m, tea.Batch(cmds...)
	}
	if c := m.selectedContainer(); c != nil {
		if c.State == "running" {
			return m, m.stopContainer(c.ID, c.Name)
		}
		return m, m.startContainer(c.ID, c.Name)
	}
	return m, nil
}

func (m Model) doRestart() (tea.Model, tea.Cmd) {
	if len(m.selected) > 0 {
		var cmds []tea.Cmd
		for _, c := range m.containers {
			if m.selected[c.ID] {
				cmds = append(cmds, m.restartContainer(c.ID, c.Name))
			}
		}
		m.selected = make(map[string]bool)
		return m, tea.Batch(cmds...)
	}
	if c := m.selectedContainer(); c != nil {
		return m, m.restartContainer(c.ID, c.Name)
	}
	return m, nil
}

func (m Model) doPauseUnpause() (tea.Model, tea.Cmd) {
	if len(m.selected) > 0 {
		var cmds []tea.Cmd
		for _, c := range m.containers {
			if m.selected[c.ID] {
				switch c.State {
				case "running":
					cmds = append(cmds, m.pauseContainer(c.ID, c.Name))
				case "paused":
					cmds = append(cmds, m.unpauseContainer(c.ID, c.Name))
				}
			}
		}
		m.selected = make(map[string]bool)
		if len(cmds) == 0 {
			m.notify("Only running or paused containers can be toggled", true)
			return m, nil
		}
		return m, tea.Batch(cmds...)
	}
	if c := m.selectedContainer(); c != nil {
		switch c.State {
		case "running":
			return m, m.pauseContainer(c.ID, c.Name)
		case "paused":
			return m, m.unpauseContainer(c.ID, c.Name)
		default:
			m.notify("Container must be running or paused", true)
		}
	}
	return m, nil
}

func (m Model) doKill() (tea.Model, tea.Cmd) {
	if len(m.selected) > 0 {
		var cmds []tea.Cmd
		for _, c := range m.containers {
			if m.selected[c.ID] && (c.State == "running" || c.State == "paused") {
				cmds = append(cmds, m.killContainer(c.ID, c.Name))
			}
		}
		m.selected = make(map[string]bool)
		if len(cmds) == 0 {
			m.notify("Only running or paused containers can be killed", true)
			return m, nil
		}
		return m, tea.Batch(cmds...)
	}
	if c := m.selectedContainer(); c != nil {
		if c.State == "running" || c.State == "paused" {
			return m, m.killContainer(c.ID, c.Name)
		}
		m.notify("Container must be running or paused", true)
	}
	return m, nil
}

func (m Model) confirmRemove() (tea.Model, tea.Cmd) {
	targets := m.removeTargets()
	if len(targets) == 0 {
		return m, nil
	}
	names := make([]string, 0, len(targets))
	for _, c := range targets {
		names = append(names, c.Name)
	}
	msg := fmt.Sprintf("Remove %d container(s)?\n\n  %s\n\nThis cannot be undone.", len(names), strings.Join(names, ", "))
	m.dialog = dialogConfirm
	m.confirmMsg = msg
	m.confirmOK = m.buildRemoveCmd(targets)
	return m, nil
}

func (m Model) buildRemoveCmd(targets []docker.ContainerInfo) tea.Cmd {
	var cmds []tea.Cmd
	for _, c := range targets {
		id, name := c.ID, c.Name
		cmds = append(cmds, func() tea.Msg {
			if err := m.client.RemoveContainer(id, true); err != nil {
				return errMsg{err}
			}
			return actionDoneMsg{"Removed", name}
		})
	}
	return tea.Batch(cmds...)
}

func (m Model) buildDetailRemoveCmd(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveContainer(id, true); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Removed", name}
	}
}

func (m Model) buildImageRemoveCmd(id, tag string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveImage(id, false); err != nil {
			return errMsg{err}
		}
		return imageActionDoneMsg{"Removed image", tag}
	}
}

func (m Model) removeTargets() []docker.ContainerInfo {
	if len(m.selected) > 0 {
		var out []docker.ContainerInfo
		for _, c := range m.containers {
			if m.selected[c.ID] {
				out = append(out, c)
			}
		}
		return out
	}
	if c := m.selectedContainer(); c != nil {
		return []docker.ContainerInfo{*c}
	}
	return nil
}

func (m Model) execIntoContainer() (tea.Model, tea.Cmd) {
	c := m.selectedContainer()
	if c == nil || c.State != "running" {
		m.notify("Container must be running to exec", true)
		return m, nil
	}
	return m, m.execIntoContainerCmd(c.ID)
}

// ── Filter ──────────────────────────────────────────────────────────────

func (m Model) handleFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.filtering = false
		if m.filterText == "" {
			m.cursor = 0
		}
	case "ctrl+u":
		m.filterText = ""
		m.cursor = 0
	case "backspace":
		backspaceTextInput(&m.filterText)
		m.cursor = 0
	default:
		if appendTextInput(&m.filterText, msg) {
			m.cursor = 0
		}
	}
	m.rebuildFilteredCache()
	m.clampCursorToFiltered()
	return m, nil
}

func (m Model) handleVolFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.volFiltering = false
		if m.volFilterText == "" {
			m.volCursor = 0
		}
	case "ctrl+u":
		m.volFilterText = ""
		m.volCursor = 0
	case "backspace":
		backspaceTextInput(&m.volFilterText)
		m.volCursor = 0
	default:
		if appendTextInput(&m.volFilterText, msg) {
			m.volCursor = 0
		}
	}
	m.clampVolCursorToFiltered()
	return m, nil
}

func (m Model) handleNetFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.netFiltering = false
		if m.netFilterText == "" {
			m.netCursor = 0
		}
	case "ctrl+u":
		m.netFilterText = ""
		m.netCursor = 0
	case "backspace":
		backspaceTextInput(&m.netFilterText)
		m.netCursor = 0
	default:
		if appendTextInput(&m.netFilterText, msg) {
			m.netCursor = 0
		}
	}
	m.clampNetCursorToFiltered()
	return m, nil
}

func (m Model) filteredContainers() []docker.ContainerInfo {
	key := filteredCacheKey{filter: m.filterText, sortMode: m.sortMode, n: len(m.containers)}
	if m.filteredCache != nil && m.filteredCacheKey == key {
		return m.filteredCache
	}
	return m.computeFilteredContainers()
}

func (m Model) filteredVolumes() []docker.VolumeInfo {
	if m.volFilterText == "" {
		return m.volumes
	}
	q := strings.ToLower(m.volFilterText)
	var out []docker.VolumeInfo
	for _, vol := range m.volumes {
		if strings.Contains(strings.ToLower(vol.Name), q) ||
			strings.Contains(strings.ToLower(vol.Driver), q) {
			out = append(out, vol)
		}
	}
	return out
}

func (m Model) filteredNetworks() []docker.NetworkResource {
	if m.netFilterText == "" {
		return m.networks
	}
	q := strings.ToLower(m.netFilterText)
	var out []docker.NetworkResource
	for _, net := range m.networks {
		if strings.Contains(strings.ToLower(net.Name), q) ||
			strings.Contains(strings.ToLower(net.Driver), q) {
			out = append(out, net)
		}
	}
	return out
}

func (m Model) selectedContainer() *docker.ContainerInfo {
	fc := m.filteredContainers()
	if len(fc) == 0 || m.cursor < 0 || m.cursor >= len(fc) {
		return nil
	}
	c := fc[m.cursor]
	return &c
}

// ── Dialog ──────────────────────────────────────────────────────────────

func (m Model) handleDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.dialog {
	case dialogConfirm:
		switch msg.String() {
		case "y", "enter":
			m.dialog = dialogNone
			cmd := m.confirmOK
			m.selected = make(map[string]bool)
			return m, cmd
		case "n", "esc", "q":
			m.dialog = dialogNone
		}

	case dialogTheme:
		switch msg.String() {
		case "up", "k":
			if m.themeCursor > 0 {
				m.themeCursor--
			}
		case "down", "j":
			if m.themeCursor < len(config.Themes)-1 {
				m.themeCursor++
			}
		case "enter":
			t := &config.Themes[m.themeCursor]
			applyTheme(t)
			m.cfg.Theme = t.Name
			go config.Save(m.cfg)
			m.dialog = dialogNone
		case "esc", "q":
			m.dialog = dialogNone
		}

	case dialogInput:
		switch msg.String() {
		case "enter":
			text := m.inputText
			m.inputText = ""
			m.dialog = dialogNone
			if m.inputSubmit != nil {
				return m, m.inputSubmit(text)
			}
		case "esc":
			m.inputText = ""
			m.dialog = dialogNone
		case "backspace":
			backspaceTextInput(&m.inputText)
		default:
			if appendTextInput(&m.inputText, msg) {
			}
		}

	case dialogHelp:
		switch msg.String() {
		case "up", "k":
			if m.helpScroll > 0 {
				m.helpScroll--
			}
		case "down", "j":
			if m.helpScroll < m.helpDialogMaxScroll() {
				m.helpScroll++
			}
		case "pgup":
			m.helpScroll = max(0, m.helpScroll-m.helpDialogMaxVisible()/2)
		case "pgdown":
			m.helpScroll = min(m.helpDialogMaxScroll(), m.helpScroll+m.helpDialogMaxVisible()/2)
		case "home", "g":
			m.helpScroll = 0
		case "end", "G":
			m.helpScroll = m.helpDialogMaxScroll()
		case "esc", "q", "?":
			m.dialog = dialogNone
			m.helpScroll = 0
		}

	case dialogCommandPalette:
		switch msg.String() {
		case "esc":
			m.dialog = dialogNone
			m.commandPaletteText = ""
			m.commandPaletteResults = nil
		case "enter":
			if m.commandPaletteCursor < len(m.commandPaletteResults) {
				run := m.commandPaletteResults[m.commandPaletteCursor].Run
				m.dialog = dialogNone
				m.commandPaletteText = ""
				m.commandPaletteResults = nil
				if run != nil {
					var cmd tea.Cmd
					m, cmd = run(m)
					return m, cmd
				}
			}
		case "up", "ctrl+p":
			if m.commandPaletteCursor > 0 {
				m.commandPaletteCursor--
			}
		case "down", "ctrl+n":
			if m.commandPaletteCursor < len(m.commandPaletteResults)-1 {
				m.commandPaletteCursor++
			}
		case "backspace":
			backspaceTextInput(&m.commandPaletteText)
			m.commandPaletteCursor = 0
			m.commandPaletteResults = m.filterCommands(m.commandPaletteText)
		default:
			if appendTextInput(&m.commandPaletteText, msg) {
				m.commandPaletteCursor = 0
				m.commandPaletteResults = m.filterCommands(m.commandPaletteText)
			}
		}
	}
	return m, nil
}

// ── Detail ──────────────────────────────────────────────────────────────

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.detailTab == tabTerminal && m.terminalActive {
		if !m.terminalInputFocused && isTerminalPrintableKey(msg) {
			m.terminalInputFocused = true
		}
		if m.terminalInputFocused {
			return m.handleDetailTerminalInput(msg)
		}
	}

	switch msg.String() {
	case "esc":
		if m.detailTab == tabLogs && len(m.logViewer.Selection) > 0 {
			m.logViewer.ClearSelection()
			return m, nil
		}
		m.stopLogStreaming()
		m.stopTerminalSession()
		m.terminalInputFocused = false
		m.view = viewList
		m.inspected = nil
		return m, m.refreshContainers()
	case "h", "backspace":
		if m.detailTab == tabTerminal && m.terminalActive {
			return m, nil
		}
		if m.detailTab == tabLogs && len(m.logViewer.Selection) > 0 {
			m.logViewer.ClearSelection()
			return m, nil
		}
		m.stopLogStreaming()
		m.stopTerminalSession()
		m.terminalInputFocused = false
		m.view = viewList
		m.inspected = nil
		return m, m.refreshContainers()
	case "q":
		return m.quit()
	case "tab", "right":
		m.detailTab = (m.detailTab + 1) % tabCount
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "shift+tab", "left":
		m.detailTab = (m.detailTab + tabCount - 1) % tabCount
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "1":
		m.detailTab = tabInfo
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "2":
		m.detailTab = tabResources
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "3":
		m.detailTab = tabEnv
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "4":
		m.detailTab = tabLogs
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "5":
		m.detailTab = tabTerminal
		m.detailScroll = 0
		return m.onTabSwitch()
	case "6":
		m.detailTab = tabDiff
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "7":
		m.detailTab = tabProcesses
		m.detailScroll = 0
		m.terminalInputFocused = false
		return m.onTabSwitch()
	case "?":
		m.dismissHint()
		m.dialog = dialogHelp
		m.helpScroll = 0
		return m, nil
	}

	if m.detailTab == tabLogs {
		rows := m.detailLogContentRows()
		switch msg.String() {
		case "up", "k":
			m.logViewer.MoveFocus(-1, rows)
			return m, nil
		case "down", "j":
			m.logViewer.MoveFocus(1, rows)
			return m, nil
		case "pgup":
			m.logViewer.ScrollPage(-1, rows)
			return m, nil
		case "pgdown":
			m.logViewer.ScrollPage(1, rows)
			return m, nil
		case "home":
			m.logViewer.ScrollHome(rows)
			m.logViewer.Focused = 0
			return m, nil
		case "end":
			m.logViewer.ScrollEnd()
			filtered := m.logViewer.FilteredEntries()
			if len(filtered) > 0 {
				m.logViewer.Focused = len(filtered) - 1
			}
			return m, nil
		case " ":
			m.logViewer.ToggleSelectFocused()
			return m, nil
		case "V":
			m.logViewer.SelectRangeToFocused()
			return m, nil
		case "y":
			return m.copyLogViewerLines(&m.logViewer)
		case "L":
			m.logViewer.ShowLegend = !m.logViewer.ShowLegend
			return m, nil
		}
	}

	if m.detailTab == tabTerminal && !m.terminalInputFocused {
		switch msg.String() {
		case "ctrl+\\":
			m.stopTerminalSession()
			m.notify("Terminal detached", false)
			return m, nil
		case "i", "enter":
			m.terminalInputFocused = true
			return m, nil
		case "x":
			if m.inspected != nil && m.inspected.State == "running" && !m.terminalActive {
				return m, m.startTerminal(m.inspected.ID)
			}
			return m, nil
		}
	}

	if !(m.detailTab == tabTerminal && m.terminalInputFocused) {
		switch msg.String() {
		case "up", "k":
			if m.detailTab == tabTerminal {
				m.terminalFollow = false
			}
			if m.detailScroll > 0 {
				m.detailScroll--
			}
			if m.detailTab == tabTerminal {
				m.syncTerminalScroll()
			}
		case "down", "j":
			if m.detailTab == tabTerminal {
				if !m.terminalFollow {
					m.detailScroll++
				}
				m.syncTerminalScroll()
			} else {
				m.detailScroll++
			}
		case "pgup":
			step := detailPageStep(m.height)
			if m.detailTab == tabTerminal {
				m.terminalFollow = false
			}
			m.detailScroll = max(0, m.detailScroll-step)
			if m.detailTab == tabTerminal {
				m.syncTerminalScroll()
			}
		case "pgdown":
			step := detailPageStep(m.height)
			if m.detailTab == tabTerminal {
				if !m.terminalFollow {
					m.detailScroll += step
				}
				m.syncTerminalScroll()
			} else {
				m.detailScroll += step
			}
		case "home":
			if m.detailTab == tabTerminal {
				m.terminalFollow = false
			}
			m.detailScroll = 0
			if m.detailTab == tabTerminal {
				m.syncTerminalScroll()
			}
		case "end":
			if m.detailTab == tabTerminal {
				m.terminalFollow = true
			}
			m.detailScroll = 1 << 20
			if m.detailTab == tabTerminal {
				m.syncTerminalScroll()
			}
		}
	}

	if m.detailAllowsContainerActions() {
		switch msg.String() {
		case "s":
			if m.inspected != nil {
				if m.inspected.State == "running" {
					return m, m.stopContainer(m.inspected.ID, m.inspected.Name)
				}
				return m, m.startContainer(m.inspected.ID, m.inspected.Name)
			}
		case "R":
			if m.inspected != nil {
				return m, m.restartContainer(m.inspected.ID, m.inspected.Name)
			}
		case "P":
			if m.inspected != nil {
				switch m.inspected.State {
				case "running":
					return m, m.pauseContainer(m.inspected.ID, m.inspected.Name)
				case "paused":
					return m, m.unpauseContainer(m.inspected.ID, m.inspected.Name)
				default:
					m.notify("Container must be running or paused", true)
				}
			}
		case "K":
			if m.inspected != nil {
				if m.inspected.State == "running" || m.inspected.State == "paused" {
					return m, m.killContainer(m.inspected.ID, m.inspected.Name)
				}
				m.notify("Container must be running or paused", true)
			}
		case "d":
			if m.inspected != nil {
				c := m.inspected
				m.dialog = dialogConfirm
				m.confirmMsg = fmt.Sprintf("Remove container %q?\n\nThis cannot be undone.", c.Name)
				m.confirmOK = m.buildDetailRemoveCmd(c.ID, c.Name)
				m.view = viewList
			}
		case "e":
			if m.inspected != nil && m.inspected.State == "running" {
				return m, m.execIntoContainerCmd(m.inspected.ID)
			}
		}
	}

	switch msg.String() {
	case "l":
		if m.detailTab == tabLogs && m.inspected != nil {
			if m.liveLogging {
				m.stopLogStreaming()
			} else {
				return m, m.streamLogs(m.inspected.ID)
			}
		}
	case "E":
		if m.detailTab == tabLogs {
			return m.promptExportDetailLogs()
		}
	case "f":
		if m.detailTab == tabDiff && m.inspected != nil {
			return m, m.getDiff(m.inspected.ID)
		}
	case "p":
		if m.detailTab == tabProcesses && m.inspected != nil {
			return m, m.getTop(m.inspected.ID)
		}
	case "t":
		m.dialog = dialogTheme
	}
	return m, nil
}

func (m Model) onTabSwitch() (tea.Model, tea.Cmd) {
	if m.detailTab != tabLogs {
		m.stopLogStreaming()
	}
	if m.detailTab == tabProcesses && m.inspected != nil {
		m.processLoaded = false
		return m, m.getTop(m.inspected.ID)
	}
	if m.detailTab != tabTerminal {
		m.stopTerminalSession()
		m.terminalInputFocused = false
	} else {
		m.terminalFollow = true
		if m.terminalActive {
			m.terminalInputFocused = true
		} else {
			m.terminalInputFocused = false
		}
		m.syncTerminalScroll()
	}
	if m.detailTab == tabTerminal && m.inspected != nil && m.inspected.State == "running" && !m.terminalActive {
		return m, m.startTerminal(m.inspected.ID)
	}
	return m, nil
}

func (m Model) updateCentralLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.centralLogFiltering {
		switch msg.String() {
		case "esc", "enter":
			m.centralLogFiltering = false
		case "ctrl+u":
			m.centralLogFilter = ""
			m.centralLogRegex = false
			m.centralLogs.SetFilter("")
			m.centralLogs.SetUseRegex(false)
		case "r":
			m.centralLogRegex = !m.centralLogRegex
			m.centralLogs.SetUseRegex(m.centralLogRegex)
		case "backspace":
			backspaceTextInput(&m.centralLogFilter)
			m.centralLogs.SetFilter(m.centralLogFilter)
		default:
			if appendTextInput(&m.centralLogFilter, msg) {
				m.centralLogs.SetFilter(m.centralLogFilter)
			}
		}
		return m, nil
	}

	height := m.centralLogContentRows()
	switch msg.String() {
	case "esc", "h":
		if len(m.centralLogs.Selection) > 0 {
			m.centralLogs.ClearSelection()
			return m, nil
		}
		return m.leaveCentralLogs()
	case "backspace":
		return m.leaveCentralLogs()
	case "q":
		return m.quit()
	case "/":
		m.centralLogFiltering = true
		m.centralLogFilter = ""
		m.centralLogs.SetFilter("")
	case "up", "k":
		m.centralLogs.MoveFocus(-1, height)
	case "down", "j":
		m.centralLogs.MoveFocus(1, height)
	case "pgup":
		m.centralLogs.ScrollPage(-1, height)
	case "pgdown":
		m.centralLogs.ScrollPage(1, height)
	case "home":
		m.centralLogs.ScrollHome(height)
		m.centralLogs.Focused = 0
	case "end":
		m.centralLogs.ScrollEnd()
		filtered := m.centralLogs.FilteredEntries()
		if len(filtered) > 0 {
			m.centralLogs.Focused = len(filtered) - 1
		}
	case " ":
		m.centralLogs.ToggleSelectFocused()
	case "V":
		m.centralLogs.SelectRangeToFocused()
	case "a":
		m.centralLogs.ShowAllContainers()
	case "L":
		m.centralLogs.ShowLegend = !m.centralLogs.ShowLegend
	case "y":
		return m.copyLogViewerLines(&m.centralLogs)
	case "E":
		return m.promptExportCentralLogs()
	default:
		if idx, ok := centralLogTargetIndex(msg.String()); ok && idx < len(m.centralLogTargets) {
			m.centralLogs.ToggleContainer(m.centralLogTargets[idx].ID)
		}
	}
	return m, nil
}

func centralLogTargetIndex(key string) (int, bool) {
	if len(key) != 1 || key[0] < '1' || key[0] > '9' {
		return 0, false
	}
	return int(key[0] - '1'), true
}

func (m Model) copyLogViewerLines(viewer *LogViewerState) (tea.Model, tea.Cmd) {
	entries := viewer.SelectedEntries()
	if len(entries) == 0 {
		if entry, ok := viewer.FocusedEntry(); ok {
			entries = []LogEntry{entry}
		}
	}
	if len(entries) == 0 {
		m.notify("No log line to copy", true)
		return m, nil
	}
	if err := CopyLogEntries(entries); err != nil {
		m.notify("Failed to copy: "+err.Error(), true)
	} else if len(entries) == 1 {
		m.notify("Copied to clipboard", false)
	} else {
		m.notify(fmt.Sprintf("Copied %d lines", len(entries)), false)
	}
	return m, nil
}

func (m Model) promptExportCentralLogs() (tea.Model, tea.Cmd) {
	m.dialog = dialogInput
	m.inputPrompt = "Export logs to file:"
	m.inputText = ""
	m.inputSubmit = func(path string) tea.Cmd {
		return m.exportCentralLogs(path)
	}
	return m, nil
}

func (m Model) exportCentralLogs(path string) tea.Cmd {
	entries := m.centralLogs.Entries
	if selected := m.centralLogs.SelectedEntries(); len(selected) > 0 {
		entries = selected
	} else {
		entries = m.centralLogs.FilteredEntries()
	}
	return func() tea.Msg {
		var lines []string
		for _, entry := range entries {
			line := ""
			if !entry.Timestamp.IsZero() {
				line = entry.Timestamp.Format("2006-01-02 15:04:05") + " "
			}
			if len(m.centralLogTargets) > 1 {
				name := entry.ContainerName
				if name == "" {
					name = entry.ContainerID
				}
				line += "[" + name + "] "
			}
			line += entry.Message
			lines = append(lines, line)
		}
		content := strings.Join(lines, "\n")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Exported", fmt.Sprintf("%d lines to %s", len(lines), path)}
	}
}

func (m Model) promptExportDetailLogs() (tea.Model, tea.Cmd) {
	m.dialog = dialogInput
	m.inputPrompt = "Export logs to file:"
	m.inputText = ""
	m.inputSubmit = func(path string) tea.Cmd {
		return m.exportDetailLogs(path)
	}
	return m, nil
}

func (m Model) exportDetailLogs(path string) tea.Cmd {
	entries := m.logViewer.Entries
	if selected := m.logViewer.SelectedEntries(); len(selected) > 0 {
		entries = selected
	}
	return func() tea.Msg {
		var lines []string
		for _, entry := range entries {
			line := ""
			if !entry.Timestamp.IsZero() {
				line = entry.Timestamp.Format("2006-01-02 15:04:05") + " "
			}
			line += entry.Message
			lines = append(lines, line)
		}
		content := strings.Join(lines, "\n")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Exported", fmt.Sprintf("%d lines to %s", len(lines), path)}
	}
}

func (m *Model) syncTerminalScroll() {
	if m.detailTab != tabTerminal || m.inspected == nil {
		return
	}
	boxWidth := max(m.width-4, 30)
	contentWidth := max(boxWidth-6, 24)
	tabContent := m.renderTerminalTab(m.inspected, contentWidth)
	lines := strings.Split(tabContent, "\n")
	availHeight := m.detailBoxInnerHeight()
	maxScroll := max(0, len(lines)-availHeight)
	m.detailScroll, m.terminalFollow = normalizeTerminalScroll(m.detailScroll, maxScroll, m.terminalFollow)
}

func (m *Model) handleContainerStateTransitions(prev []docker.ContainerInfo, next []docker.ContainerInfo) {
	prevByID := make(map[string]docker.ContainerInfo, len(prev))
	for _, c := range prev {
		prevByID[c.ID] = c
	}
	nextByID := make(map[string]docker.ContainerInfo, len(next))
	for _, c := range next {
		nextByID[c.ID] = c
	}
	if m.inspected != nil {
		if cur, ok := nextByID[m.inspected.ID]; ok {
			m.inspected.State = cur.State
			m.inspected.Status = cur.Status
		}
	}
	if m.terminalActive && m.inspected != nil {
		cur, okNow := nextByID[m.inspected.ID]
		prevC, okPrev := prevByID[m.inspected.ID]
		if !okNow || cur.State != "running" || (okPrev && prevC.State == "running" && cur.State != "running") {
			m.stopTerminalSession()
			m.terminalOutput += "\n[terminal closed: container no longer running]\n"
			m.notify("Terminal closed: container is not running", true)
		}
	}
}

// ── Images ──────────────────────────────────────────────────────────────

func (m Model) updateImages(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		m.view = viewList
	case "up", "k":
		if m.imgCursor > 0 {
			m.imgCursor--
		}
	case "down", "j":
		if m.imgCursor < len(m.images)-1 {
			m.imgCursor++
		}
	case "d":
		if m.imgCursor < len(m.images) {
			img := m.images[m.imgCursor]
			tag := img.DisplayTag()
			m.dialog = dialogConfirm
			m.confirmMsg = fmt.Sprintf("Remove image %q?\n\nThis cannot be undone.", tag)
			id := img.ID
			m.confirmOK = m.buildImageRemoveCmd(id, tag)
		}
	case "p":
		m.loading = true
		m.dialog = dialogInput
		m.inputPrompt = "Pull image (e.g. nginx:latest):"
		m.inputText = ""
		m.inputSubmit = func(ref string) tea.Cmd {
			return m.pullImage(ref)
		}
	case "P":
		m.dialog = dialogConfirm
		m.confirmMsg = "Prune dangling images?\n\nThis removes untagged images not referenced by containers."
		m.loading = true
		m.confirmOK = m.pruneDanglingImages()
	case "r":
		m.loading = true
		return m, m.fetchImages()
	case "t":
		m.dialog = dialogTheme
	}
	return m, nil
}

// ── Events ──────────────────────────────────────────────────────────────

func (m Model) updateEvents(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		if m.eventsCancel != nil {
			m.eventsCancel()
			m.eventsCancel = nil
		}
		m.view = viewList
	case "c":
		m.events = nil
	case "up", "k":
		if m.eventsCursor > 0 {
			m.eventsCursor--
		}
	case "down", "j":
		if m.eventsCursor < len(m.events)-1 {
			m.eventsCursor++
		}
	}
	return m, nil
}

// ── Volumes ─────────────────────────────────────────────────────────────

func (m Model) fetchVolumes() tea.Cmd {
	return func() tea.Msg {
		vols, err := m.client.ListVolumes()
		if err != nil {
			return errMsg{err}
		}
		return volumesMsg(vols)
	}
}

func (m Model) updateVolumes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vols := m.filteredVolumes()
	switch msg.String() {
	case "esc", "q", "h":
		m.view = viewList
		m.volFiltering = false
		m.volFilterText = ""
	case "up", "k":
		if m.volCursor > 0 {
			m.volCursor--
		}
	case "down", "j":
		if m.volCursor < len(vols)-1 {
			m.volCursor++
		}
	case "home", "g":
		m.volCursor = 0
	case "end", "G":
		if n := len(vols); n > 0 {
			m.volCursor = n - 1
		}
	case " ":
		if m.volCursor < len(vols) {
			name := vols[m.volCursor].Name
			if m.selected[name] {
				delete(m.selected, name)
			} else {
				m.selected[name] = true
			}
		}
	case "a":
		if len(m.selected) > 0 {
			m.selected = make(map[string]bool)
		} else {
			for _, vol := range vols {
				m.selected[vol.Name] = true
			}
		}
	case "d":
		return m.confirmRemoveVolumes()
	case "p":
		return m.confirmPruneVolumes()
	case "/":
		m.volFiltering = true
		m.volFilterText = ""
	case "ctrl+u":
		m.volFilterText = ""
		m.volCursor = 0
		m.clampVolCursorToFiltered()
	case "r":
		m.loading = true
		return m, m.fetchVolumes()
	case "t":
		m.dialog = dialogTheme
	}
	return m, nil
}

func (m Model) confirmRemoveVolumes() (tea.Model, tea.Cmd) {
	if len(m.selected) == 0 && m.volCursor < len(m.filteredVolumes()) {
		m.selected[m.filteredVolumes()[m.volCursor].Name] = true
	}
	if len(m.selected) == 0 {
		return m, nil
	}
	names := make([]string, 0, len(m.selected))
	for name := range m.selected {
		names = append(names, name)
	}
	msg := fmt.Sprintf("Remove %d volume(s)?\n\n  %s\n\nThis cannot be undone.", len(names), strings.Join(names, ", "))
	m.dialog = dialogConfirm
	m.confirmMsg = msg
	m.confirmOK = m.buildVolumeRemoveCmd(names)
	return m, nil
}

func (m Model) buildVolumeRemoveCmd(names []string) tea.Cmd {
	var cmds []tea.Cmd
	for _, name := range names {
		n := name
		cmds = append(cmds, m.removeVolume(n))
	}
	return tea.Batch(cmds...)
}

func (m Model) confirmPruneVolumes() (tea.Model, tea.Cmd) {
	m.dialog = dialogConfirm
	m.confirmMsg = "Remove all orphaned volumes?\n\nThis cannot be undone."
	m.confirmOK = m.pruneVolumesCmd()
	return m, nil
}

// ── Networks ─────────────────────────────────────────────────────────────

func (m Model) fetchNetworks() tea.Cmd {
	return func() tea.Msg {
		nets, err := m.client.ListNetworks()
		if err != nil {
			return errMsg{err}
		}
		return networksMsg(nets)
	}
}

func (m Model) updateNetworks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	nets := m.filteredNetworks()
	switch msg.String() {
	case "esc", "q", "h":
		m.view = viewList
		m.netFiltering = false
		m.netFilterText = ""
	case "up", "k":
		if m.netCursor > 0 {
			m.netCursor--
		}
	case "down", "j":
		if m.netCursor < len(nets)-1 {
			m.netCursor++
		}
	case "home", "g":
		m.netCursor = 0
	case "end", "G":
		if n := len(nets); n > 0 {
			m.netCursor = n - 1
		}
	case " ":
		if m.netCursor < len(nets) {
			id := nets[m.netCursor].ID
			if m.selected[id] {
				delete(m.selected, id)
			} else {
				m.selected[id] = true
			}
		}
	case "a":
		if len(m.selected) > 0 {
			m.selected = make(map[string]bool)
		} else {
			for _, net := range nets {
				m.selected[net.ID] = true
			}
		}
	case "d":
		return m.confirmRemoveNetworks()
	case "/":
		m.netFiltering = true
		m.netFilterText = ""
	case "ctrl+u":
		m.netFilterText = ""
		m.netCursor = 0
		m.clampNetCursorToFiltered()
	case "r":
		m.loading = true
		return m, m.fetchNetworks()
	case "t":
		m.dialog = dialogTheme
	}
	return m, nil
}

func (m Model) confirmRemoveNetworks() (tea.Model, tea.Cmd) {
	nets := m.filteredNetworks()
	if len(m.selected) == 0 && m.netCursor < len(nets) {
		m.selected[nets[m.netCursor].ID] = true
	}
	if len(m.selected) == 0 {
		return m, nil
	}
	ids := make([]string, 0, len(m.selected))
	names := make([]string, 0, len(m.selected))
	for id := range m.selected {
		ids = append(ids, id)
		for _, net := range m.networks {
			if net.ID == id {
				names = append(names, net.Name)
				break
			}
		}
	}
	msg := fmt.Sprintf("Remove %d network(s)?\n\n  %s\n\nThis cannot be undone.", len(names), strings.Join(names, ", "))
	m.dialog = dialogConfirm
	m.confirmMsg = msg
	m.confirmOK = m.buildNetworkRemoveCmd(ids)
	return m, nil
}

func (m Model) buildNetworkRemoveCmd(ids []string) tea.Cmd {
	var cmds []tea.Cmd
	for _, id := range ids {
		n := id
		cmds = append(cmds, m.removeNetwork(n))
	}
	return tea.Batch(cmds...)
}

func (m Model) removeNetwork(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RemoveNetwork(id); err != nil {
			return errMsg{err}
		}
		for _, net := range m.networks {
			if net.ID == id {
				return networkActionDoneMsg{"Removed", net.Name}
			}
		}
		return networkActionDoneMsg{"Removed", id}
	}
}

func (m Model) updateNotifications(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "h":
		m.view = viewList
	case "up", "k":
		if m.notifyCursor > 0 {
			m.notifyCursor--
		}
	case "down", "j":
		if m.notifyCursor < len(m.notifyHistory)-1 {
			m.notifyCursor++
		}
	case "home", "g":
		m.notifyCursor = 0
	case "end", "G":
		if n := len(m.notifyHistory); n > 0 {
			m.notifyCursor = n - 1
		}
	case "c":
		m.notifyHistory = nil
		m.notifyCursor = 0
	}
	return m, nil
}
