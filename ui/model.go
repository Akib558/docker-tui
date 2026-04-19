package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akib/docker-tui/config"
	"github.com/akib/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Update ──────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
		m.clampCursorToFiltered()
		return m, nil

	case imagesMsg:
		m.images = []docker.ImageInfo(msg)
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
		m.logLines = strings.Split(cleanDockerLogs(string(msg)), "\n")
		return m, nil

	case logStreamStartMsg:
		m.logCancel = msg.cancel
		m.liveLogging = true
		return m, msg.next

	case logLineMsg:
		m.logLines = append(m.logLines, msg.line)
		if len(m.logLines) > 500 {
			m.logLines = m.logLines[len(m.logLines)-500:]
		}
		if m.view == viewDetail && m.detailTab == tabLogs && m.liveLogging {
			return m, msg.next
		}
		return m, nil

	case logStreamDoneMsg:
		m.liveLogging = false
		return m, nil

	case terminalStartMsg:
		if m.terminalCancel != nil {
			m.terminalCancel()
		}
		m.terminalCancel = msg.cancel
		m.terminalWriter = msg.writer
		m.terminalActive = true
		m.terminalShell = msg.shell
		if m.terminalOutput == "" {
			m.terminalOutput = fmt.Sprintf("Connected to shell: %s\n", msg.shell)
		}
		return m, msg.next

	case terminalChunkMsg:
		m.terminalOutput += msg.chunk
		if len(m.terminalOutput) > terminalBufferMax {
			m.terminalOutput = m.terminalOutput[len(m.terminalOutput)-terminalBufferMax:]
		}
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

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.notify(fmt.Sprintf("Error: %v", msg.err), true)
		return m, nil

	case actionDoneMsg:
		m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
		return m, m.refreshContainers()

	case imageActionDoneMsg:
		m.notify(fmt.Sprintf("%s: %s", msg.action, msg.name), false)
		return m, m.fetchImages()

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
			if m.view == viewList || m.view == viewDetail {
				cmds = append(cmds, m.refreshContainers())
			}
			if !m.fetchStats {
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
		}
		if m.dialog != dialogNone {
			return m.handleDialog(msg)
		}
		if m.filtering {
			return m.handleFilter(msg)
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
		}
	}
	return m, nil
}

// ── Mouse ───────────────────────────────────────────────────────────────

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.view == viewDetail {
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		} else {
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case tea.MouseButtonWheelDown:
		if m.view == viewDetail {
			m.detailScroll++
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
		m.view = viewEvents
		if m.eventsCancel == nil {
			return m, m.startEventStream()
		}
	case "/":
		m.filtering = true
		m.filterText = ""
	case "C":
		m.filtering = false
		m.filterText = ""
		m.cursor = 0
	case "c":
		m.groupByCompose = !m.groupByCompose
	case "t":
		m.dialog = dialogTheme
	case "+":
		if m.cfg.RefreshSeconds > 1 {
			m.cfg.RefreshSeconds--
		} else {
			m.cfg.RefreshSeconds = 1
		}
		m.refreshInterval = time.Duration(m.cfg.RefreshSeconds) * time.Second
		go config.Save(m.cfg)
	case "-":
		if m.cfg.RefreshSeconds < 30 {
			m.cfg.RefreshSeconds++
		}
		m.refreshInterval = time.Duration(m.cfg.RefreshSeconds) * time.Second
		go config.Save(m.cfg)
	}
	m.clampCursorToFiltered()
	return m, nil
}

func (m Model) openDetail(c docker.ContainerInfo) (tea.Model, tea.Cmd) {
	m.view = viewDetail
	m.detailScroll = 0
	m.detailTab = tabInfo
	m.logLines = nil
	m.diff = nil
	m.terminalInput = ""
	m.terminalOutput = ""
	m.terminalShell = ""
	m.stopLogStreaming()
	m.stopTerminalSession()
	m.loading = true
	return m, tea.Batch(m.inspectContainer(c.ID), m.fetchLogs(c.ID))
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
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.cursor = 0
		}
	default:
		if len(msg.String()) == 1 {
			m.filterText += msg.String()
			m.cursor = 0
		}
	}
	m.clampCursorToFiltered()
	return m, nil
}

func (m Model) filteredContainers() []docker.ContainerInfo {
	if m.filterText == "" {
		return m.containers
	}
	q := strings.ToLower(m.filterText)
	var out []docker.ContainerInfo
	for _, c := range m.containers {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.Image), q) ||
			strings.Contains(strings.ToLower(c.State), q) {
			out = append(out, c)
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
			if len(m.inputText) > 0 {
				m.inputText = m.inputText[:len(m.inputText)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.inputText += msg.String()
			}
		}
	}
	return m, nil
}

// ── Detail ──────────────────────────────────────────────────────────────

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.detailTab == tabTerminal {
		switch msg.String() {
		case "ctrl+\\":
			m.stopTerminalSession()
			m.notify("Terminal detached", false)
			return m, nil
		case "enter":
			if m.terminalActive {
				line := m.terminalInput
				m.terminalInput = ""
				return m, m.sendTerminalInput(line + "\n")
			}
		case "backspace":
			if len(m.terminalInput) > 0 {
				m.terminalInput = m.terminalInput[:len(m.terminalInput)-1]
				return m, nil
			}
		default:
			if len(msg.String()) == 1 {
				m.terminalInput += msg.String()
				return m, nil
			}
		}
	}

	switch msg.String() {
	case "esc", "h", "backspace":
		m.stopLogStreaming()
		m.stopTerminalSession()
		m.view = viewList
		m.inspected = nil
		return m, m.refreshContainers()
	case "q":
		return m.quit()
	case "tab", "right":
		m.detailTab = (m.detailTab + 1) % tabCount
		m.detailScroll = 0
		return m.onTabSwitch()
	case "shift+tab", "left":
		m.detailTab = (m.detailTab + tabCount - 1) % tabCount
		m.detailScroll = 0
		return m.onTabSwitch()
	case "up", "k":
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case "down", "j":
		m.detailScroll++
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
	case "d":
		if m.inspected != nil {
			c := m.inspected
			m.dialog = dialogConfirm
			m.confirmMsg = fmt.Sprintf("Remove container %q?\n\nThis cannot be undone.", c.Name)
			m.confirmOK = func() tea.Msg {
				if err := m.client.RemoveContainer(c.ID, true); err != nil {
					return errMsg{err}
				}
				return actionDoneMsg{"Removed", c.Name}
			}
			m.view = viewList
		}
	case "e":
		if m.inspected != nil && m.inspected.State == "running" {
			return m, m.execIntoContainerCmd(m.inspected.ID)
		}
	case "l":
		if m.detailTab == tabLogs && m.inspected != nil {
			if m.liveLogging {
				m.stopLogStreaming()
			} else {
				return m, m.streamLogs(m.inspected.ID)
			}
		}
	case "x":
		if m.detailTab == tabTerminal && m.inspected != nil && m.inspected.State == "running" && !m.terminalActive {
			return m, m.startTerminal(m.inspected.ID)
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
	if m.detailTab != tabTerminal {
		m.stopTerminalSession()
	}
	if m.detailTab == tabTerminal && m.inspected != nil && m.inspected.State == "running" && !m.terminalActive {
		return m, m.startTerminal(m.inspected.ID)
	}
	return m, nil
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
			m.confirmOK = func() tea.Msg {
				if err := m.client.RemoveImage(id, false); err != nil {
					return errMsg{err}
				}
				return imageActionDoneMsg{"Removed image", tag}
			}
		}
	case "p":
		m.dialog = dialogInput
		m.inputPrompt = "Pull image (e.g. nginx:latest):"
		m.inputText = ""
		m.inputSubmit = func(ref string) tea.Cmd {
			return m.pullImage(ref)
		}
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
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.events)-1 {
			m.cursor++
		}
	}
	return m, nil
}
