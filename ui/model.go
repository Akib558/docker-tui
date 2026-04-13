package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/akib/docker-tui/config"
	"github.com/akib/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

// ── View / Dialog enums ─────────────────────────────────────────────────

type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewImages
	viewEvents
)

type dialogMode int

const (
	dialogNone dialogMode = iota
	dialogConfirm
	dialogTheme
	dialogInput
)

const historyLen = 60
const terminalBufferMax = 96 * 1024

// ── Model ───────────────────────────────────────────────────────────────

type Model struct {
	// Docker
	client    *docker.Client
	containers []docker.ContainerInfo
	inspected  *docker.ContainerInfo
	images     []docker.ImageInfo
	events     []docker.DockerEvent

	// Stats
	stats      map[string]*docker.ContainerResourceStats
	cpuHistory map[string][]float64
	memHistory map[string][]float64
	fetchStats bool
	alertShown map[string]bool

	// System
	systemMem  docker.SystemMemory
	systemLoad docker.SystemLoad
	overview   *docker.DockerOverview

	// Navigation
	view        viewState
	cursor      int
	imgCursor   int
	width       int
	height      int

	// Detail
	detailScroll int
	detailTab    int // 0=info 1=resources 2=env 3=logs 4=terminal
	logLines     []string
	logCancel    func()
	liveLogging  bool
	diff         []docker.DiffEntry
	terminalInput  string
	terminalOutput string
	terminalCancel func()
	terminalWriter io.Writer
	terminalActive bool
	terminalShell  string

	// Events streaming
	eventsCtx    context.Context
	eventsCancel func()

	// Filter
	filtering  bool
	filterText string

	// Multi-select
	selected map[string]bool

	// Dialog
	dialog       dialogMode
	confirmMsg   string
	confirmOK    tea.Cmd
	inputText    string
	inputPrompt  string
	inputSubmit  func(string) tea.Cmd

	// Theme
	themeCursor int

	// Notification
	notification string
	notifyIsErr  bool
	notifyTime   time.Time

	// Config / refresh
	cfg             *config.Config
	refreshInterval time.Duration

	// Meta
	loading     bool
	err         error
	startTime   time.Time
	lastRefresh time.Time

	// Compose grouping toggle
	groupByCompose bool
}

// ── Messages ────────────────────────────────────────────────────────────

type containersMsg []docker.ContainerInfo
type imagesMsg []docker.ImageInfo
type errMsg struct{ err error }
type inspectMsg struct{ info *docker.ContainerInfo }
type logsMsg string
type tickMsg time.Time
type actionDoneMsg struct{ action, name string }
type statsMsg struct {
	stats   map[string]*docker.ContainerResourceStats
	sysMem  docker.SystemMemory
	sysLoad docker.SystemLoad
}
type diffMsg []docker.DiffEntry
type imageActionDoneMsg struct{ action, name string }
type execDoneMsg struct{ err error }
type loadHistMsg struct {
	cpu map[string][]float64
	mem map[string][]float64
}
type logLineMsg struct {
	line string
	next tea.Cmd
}
type logStreamStartMsg struct {
	cancel func()
	next   tea.Cmd
}
type logStreamDoneMsg struct{}
type newEventMsg struct {
	ev   docker.DockerEvent
	next tea.Cmd
}
type eventStreamStartMsg struct {
	cancel func()
	next   tea.Cmd
}
type terminalStartMsg struct {
	cancel func()
	writer io.Writer
	shell  string
	next   tea.Cmd
}
type terminalChunkMsg struct {
	chunk string
	next  tea.Cmd
}
type terminalDoneMsg struct {
	err error
}

type initMsg struct {
	client     *docker.Client
	containers []docker.ContainerInfo
	overview   *docker.DockerOverview
	sysMem     docker.SystemMemory
	sysLoad    docker.SystemLoad
}

// ── Constructor ─────────────────────────────────────────────────────────

func NewModel(cfg *config.Config) Model {
	applyTheme(config.FindTheme(cfg.Theme))
	return Model{
		loading:         true,
		stats:           make(map[string]*docker.ContainerResourceStats),
		cpuHistory:      make(map[string][]float64),
		memHistory:      make(map[string][]float64),
		alertShown:      make(map[string]bool),
		selected:        make(map[string]bool),
		cfg:             cfg,
		refreshInterval: time.Duration(cfg.RefreshSeconds) * time.Second,
		startTime:       time.Now(),
		themeCursor:     config.ThemeIndex(cfg.Theme),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(initClient, m.loadHistory(), tickCmd(m.refreshInterval))
}

func initClient() tea.Msg {
	c, err := docker.NewClient()
	if err != nil {
		return errMsg{err}
	}
	containers, err := c.ListContainers()
	if err != nil {
		return errMsg{err}
	}
	overview, _ := c.GetDockerOverview()
	return initMsg{
		client:     c,
		containers: containers,
		overview:   overview,
		sysMem:     docker.GetSystemMemory(),
		sysLoad:    docker.GetSystemLoad(),
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

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
		return m, m.saveHistory()

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
		// Continue streaming if we're still on logs tab in detail view
		if m.view == viewDetail && m.detailTab == 3 && m.liveLogging {
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
		if m.view == viewDetail && m.detailTab == 4 && m.terminalActive {
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
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m.quit()
		}
		// Dialog intercepts input first
		if m.dialog != dialogNone {
			return m.handleDialog(msg)
		}
		// Filter mode intercepts keys
		if m.filtering {
			return m.handleFilter(msg)
		}
		// Per-view
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
			// Calculate which row was clicked.
			// Header(3) + dashboard(4-5) + table-header(1) = ~9 lines offset
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
		// Select/deselect all
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
	m.detailTab = 0
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
	// Bulk if anything selected
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
	m.confirmOK = func() tea.Msg { return nil } // placeholder
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
	cmd := exec.Command("docker", "exec", "-it", c.ID, "/bin/sh")
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{err}
	})
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
	if m.detailTab == 4 {
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
		m.detailTab = (m.detailTab + 1) % 5
		m.detailScroll = 0
		return m.onTabSwitch()
	case "shift+tab", "left":
		m.detailTab = (m.detailTab + 4) % 5
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
			m.view = viewList // go back after confirm
		}
	case "e":
		if m.inspected != nil && m.inspected.State == "running" {
			cmd := exec.Command("docker", "exec", "-it", m.inspected.ID, "/bin/sh")
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return execDoneMsg{err}
			})
		}
	case "l":
		// Toggle live log streaming (only on logs tab)
		if m.detailTab == 3 && m.inspected != nil {
			if m.liveLogging {
				m.stopLogStreaming()
			} else {
				return m, m.streamLogs(m.inspected.ID)
			}
		}
	case "x":
		if m.detailTab == 4 && m.inspected != nil && m.inspected.State == "running" && !m.terminalActive {
			return m, m.startTerminal(m.inspected.ID)
		}
	case "t":
		m.dialog = dialogTheme
	}
	return m, nil
}

func (m Model) onTabSwitch() (tea.Model, tea.Cmd) {
	// Stop live logs when leaving logs tab
	if m.detailTab != 3 {
		m.stopLogStreaming()
	}
	if m.detailTab != 4 {
		m.stopTerminalSession()
	}
	if m.detailTab == 4 && m.inspected != nil && m.inspected.State == "running" && !m.terminalActive {
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

// ── Commands ────────────────────────────────────────────────────────────

func (m Model) refreshContainers() tea.Cmd {
	return func() tea.Msg {
		containers, err := m.client.ListContainers()
		if err != nil {
			return errMsg{err}
		}
		return containersMsg(containers)
	}
}

func (m Model) fetchImages() tea.Cmd {
	return func() tea.Msg {
		imgs, err := m.client.ListImages()
		if err != nil {
			return errMsg{err}
		}
		return imagesMsg(imgs)
	}
}

func (m Model) collectStats() tea.Cmd {
	return func() tea.Msg {
		var ids []string
		for _, c := range m.containers {
			if c.State == "running" {
				ids = append(ids, c.ID)
			}
		}
		return statsMsg{
			stats:   m.client.GetAllContainerStats(ids),
			sysMem:  docker.GetSystemMemory(),
			sysLoad: docker.GetSystemLoad(),
		}
	}
}

func (m Model) inspectContainer(id string) tea.Cmd {
	return func() tea.Msg {
		info, err := m.client.InspectContainer(id)
		if err != nil {
			return errMsg{err}
		}
		return inspectMsg{info}
	}
}

func (m Model) fetchLogs(id string) tea.Cmd {
	return func() tea.Msg {
		logs, err := m.client.GetContainerLogs(id, 100)
		if err != nil {
			return logsMsg("(unable to fetch logs)")
		}
		return logsMsg(logs)
	}
}

func (m Model) streamLogs(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan string, 200)

		go func() {
			defer close(ch)
			reader, err := m.client.GetContainerLogsStream(ctx, id)
			if err != nil {
				cancel()
				return
			}
			defer reader.Close()
			buf := make([]byte, 4096)
			var partial string
			for {
				n, err := reader.Read(buf)
				if n > 0 {
					data := partial + string(buf[:n])
					lines := strings.Split(data, "\n")
					for i, line := range lines {
						if i == len(lines)-1 {
							partial = line
						} else {
							// Strip Docker log header (8 bytes)
							if len(line) > 8 && (line[0] == 1 || line[0] == 2) {
								line = line[8:]
							}
							select {
							case ch <- line:
							case <-ctx.Done():
								return
							}
						}
					}
				}
				if err != nil {
					break
				}
			}
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			line, ok := <-ch
			if !ok {
				return logStreamDoneMsg{}
			}
			return logLineMsg{line: line, next: readNext}
		}
		return logStreamStartMsg{cancel: cancel, next: readNext}
	}
}

func (m Model) startTerminal(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		started := false
		var attach io.ReadWriteCloser
		var shell string

		for _, sh := range []string{"/bin/bash", "/bin/sh"} {
			resp, err := m.client.StartContainerExecShell(ctx, id, sh)
			if err == nil {
				attach = resp
				shell = sh
				started = true
				break
			}
		}
		if !started {
			cancel()
			return terminalDoneMsg{err: fmt.Errorf("unable to start shell (/bin/bash or /bin/sh)")}
		}

		ch := make(chan string, 256)
		go func() {
			defer close(ch)
			defer attach.Close()
			buf := make([]byte, 4096)
			for {
				n, err := attach.Read(buf)
				if n > 0 {
					ch <- string(buf[:n])
				}
				if err != nil {
					return
				}
			}
		}()

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			select {
			case <-ctx.Done():
				return terminalDoneMsg{}
			case chunk, ok := <-ch:
				if !ok {
					return terminalDoneMsg{}
				}
				return terminalChunkMsg{chunk: chunk, next: readNext}
			}
		}

		stop := func() {
			cancel()
			_ = attach.Close()
		}
		return terminalStartMsg{cancel: stop, writer: attach, shell: shell, next: readNext}
	}
}

func (m Model) sendTerminalInput(text string) tea.Cmd {
	return func() tea.Msg {
		if m.terminalWriter == nil {
			return terminalDoneMsg{err: fmt.Errorf("terminal not connected")}
		}
		if _, err := io.WriteString(m.terminalWriter, text); err != nil {
			return terminalDoneMsg{err: err}
		}
		return nil
	}
}

func (m Model) startEventStream() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch := m.client.StreamEvents(ctx)

		var readNext tea.Cmd
		readNext = func() tea.Msg {
			ev, ok := <-ch
			if !ok {
				return nil
			}
			return newEventMsg{ev: ev, next: readNext}
		}
		// Store cancel via a side-channel message
		return eventStreamStartMsg{cancel: cancel, next: readNext}
	}
}

func (m Model) getDiff(id string) tea.Cmd {
	return func() tea.Msg {
		diff, err := m.client.GetContainerDiff(id)
		if err != nil {
			return diffMsg{}
		}
		return diffMsg(diff)
	}
}

func (m Model) startContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.StartContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Started", name}
	}
}

func (m Model) stopContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.StopContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Stopped", name}
	}
}

func (m Model) restartContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.RestartContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{"Restarted", name}
	}
}

func (m Model) pullImage(ref string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.PullImage(ref); err != nil {
			return errMsg{err}
		}
		return imageActionDoneMsg{"Pulled", ref}
	}
}

func (m Model) quit() (tea.Model, tea.Cmd) {
	m.stopLogStreaming()
	m.stopTerminalSession()
	if m.eventsCancel != nil {
		m.eventsCancel()
	}
	if m.client != nil {
		m.client.Close()
	}
	go config.Save(m.cfg)
	return m, tea.Quit
}

// ── Alerts ──────────────────────────────────────────────────────────────

func (m *Model) checkAlerts(id string, s *docker.ContainerResourceStats) {
	if s.CPUPercent >= m.cfg.AlertCPU && !m.alertShown[id+"_cpu"] {
		name := m.containerName(id)
		m.notify(fmt.Sprintf("⚠ HIGH CPU: %s %.0f%%", name, s.CPUPercent), true)
		m.alertShown[id+"_cpu"] = true
	} else if s.CPUPercent < m.cfg.AlertCPU*0.8 {
		delete(m.alertShown, id+"_cpu")
	}
	if s.MemPercent >= m.cfg.AlertMem && !m.alertShown[id+"_mem"] {
		name := m.containerName(id)
		m.notify(fmt.Sprintf("⚠ HIGH MEM: %s %.0f%%", name, s.MemPercent), true)
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

// ── History persistence ─────────────────────────────────────────────────

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
	cpu := m.cpuHistory
	mem := m.memHistory
	return func() tea.Msg {
		path := histPath()
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		h := histData{CPU: cpu, Mem: mem}
		data, _ := json.MarshalIndent(h, "", "")
		_ = os.WriteFile(path, data, 0644)
		return nil
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────

func (m *Model) notify(msg string, isErr bool) {
	m.notification = msg
	m.notifyIsErr = isErr
	m.notifyTime = time.Now()
}

func (m *Model) stopLogStreaming() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
	m.liveLogging = false
}

func (m *Model) stopTerminalSession() {
	if m.terminalCancel != nil {
		m.terminalCancel()
		m.terminalCancel = nil
	}
	m.terminalWriter = nil
	m.terminalActive = false
}

func (m *Model) clampCursorToFiltered() {
	fc := m.filteredContainers()
	if len(fc) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(fc) {
		m.cursor = len(fc) - 1
	}
}

func appendHist(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > historyLen {
		h = h[len(h)-historyLen:]
	}
	return h
}
