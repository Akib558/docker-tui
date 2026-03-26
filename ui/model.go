package ui

import (
	"fmt"
	"time"

	"github.com/akib/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

type view int

const (
	listView view = iota
	detailView
)

const historyLen = 60 // ~3 min at 3s ticks

type Model struct {
	// docker
	client     *docker.Client
	containers []docker.ContainerInfo
	inspected  *docker.ContainerInfo

	// resource stats
	stats      map[string]*docker.ContainerResourceStats
	cpuHistory map[string][]float64
	memHistory map[string][]float64
	fetchStats bool // guard against overlapping stats fetches

	// system
	systemMem docker.SystemMemory
	systemLoad docker.SystemLoad
	overview  *docker.DockerOverview

	// ui state
	cursor       int
	currentView  view
	width        int
	height       int
	err          error
	notification string
	notifyIsErr  bool
	notifyTime   time.Time
	detailScroll int
	detailTab    int // 0=info 1=resources 2=env 3=logs
	logs         string
	loading      bool
}

// ── Messages ────────────────────────────────────────────────────────────

type containersMsg []docker.ContainerInfo
type errMsg struct{ err error }
type inspectMsg struct{ info *docker.ContainerInfo }
type logsMsg string
type tickMsg time.Time
type actionDoneMsg struct{ action, name string }

type initMsg struct {
	client     *docker.Client
	containers []docker.ContainerInfo
	overview   *docker.DockerOverview
	sysMem     docker.SystemMemory
	sysLoad    docker.SystemLoad
}

type statsMsg struct {
	stats   map[string]*docker.ContainerResourceStats
	sysMem  docker.SystemMemory
	sysLoad docker.SystemLoad
}

// ── Constructor ─────────────────────────────────────────────────────────

func NewModel() Model {
	return Model{
		loading:    true,
		stats:      make(map[string]*docker.ContainerResourceStats),
		cpuHistory: make(map[string][]float64),
		memHistory: make(map[string][]float64),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(initClient, tickCmd())
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
	sysMem := docker.GetSystemMemory()
	sysLoad := docker.GetSystemLoad()
	return initMsg{client: c, containers: containers, overview: overview, sysMem: sysMem, sysLoad: sysLoad}
}

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ── Update ──────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case initMsg:
		m.client = msg.client
		m.containers = msg.containers
		m.overview = msg.overview
		m.systemMem = msg.sysMem
		m.systemLoad = msg.sysLoad
		m.loading = false
		// Kick off first stats collection
		m.fetchStats = true
		return m, m.collectStats()

	case containersMsg:
		m.containers = []docker.ContainerInfo(msg)
		m.loading = false
		if m.cursor >= len(m.containers) && m.cursor > 0 {
			m.cursor = len(m.containers) - 1
		}
		return m, nil

	case statsMsg:
		m.fetchStats = false
		m.stats = msg.stats
		m.systemMem = msg.sysMem
		m.systemLoad = msg.sysLoad
		for id, s := range msg.stats {
			m.cpuHistory[id] = appendHist(m.cpuHistory[id], s.CPUPercent)
			m.memHistory[id] = appendHist(m.memHistory[id], s.MemPercent)
		}
		return m, nil

	case inspectMsg:
		m.inspected = msg.info
		m.loading = false
		return m, nil

	case logsMsg:
		m.logs = string(msg)
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.notification = fmt.Sprintf("Error: %v", msg.err)
		m.notifyIsErr = true
		m.notifyTime = time.Now()
		return m, nil

	case actionDoneMsg:
		m.notification = fmt.Sprintf("%s: %s", msg.action, msg.name)
		m.notifyIsErr = false
		m.notifyTime = time.Now()
		return m, m.refreshContainers()

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd())
		if m.client != nil {
			cmds = append(cmds, m.refreshContainers())
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
		switch m.currentView {
		case listView:
			return m.updateList(msg)
		case detailView:
			return m.updateDetail(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.client != nil {
			m.client.Close()
		}
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.containers)-1 {
			m.cursor++
		}
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		if len(m.containers) > 0 {
			m.cursor = len(m.containers) - 1
		}
	case "enter", "l":
		if len(m.containers) > 0 {
			m.currentView = detailView
			m.detailScroll = 0
			m.detailTab = 0
			m.logs = ""
			m.loading = true
			c := m.containers[m.cursor]
			return m, tea.Batch(m.inspectContainer(c.ID), m.fetchLogs(c.ID))
		}
	case "s":
		if len(m.containers) > 0 {
			c := m.containers[m.cursor]
			if c.State == "running" {
				return m, m.stopContainer(c.ID, c.Name)
			}
			return m, m.startContainer(c.ID, c.Name)
		}
	case "r":
		m.loading = true
		return m, m.refreshContainers()
	}
	return m, nil
}

func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "h", "backspace":
		m.currentView = listView
		m.inspected = nil
		m.logs = ""
		return m, m.refreshContainers()
	case "q", "ctrl+c":
		if m.client != nil {
			m.client.Close()
		}
		return m, tea.Quit
	case "tab", "right":
		m.detailTab = (m.detailTab + 1) % 4
		m.detailScroll = 0
	case "shift+tab", "left":
		m.detailTab = (m.detailTab + 3) % 4
		m.detailScroll = 0
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

func (m Model) collectStats() tea.Cmd {
	return func() tea.Msg {
		var ids []string
		for _, c := range m.containers {
			if c.State == "running" {
				ids = append(ids, c.ID)
			}
		}
		stats := m.client.GetAllContainerStats(ids)
		return statsMsg{
			stats:   stats,
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
		logs, err := m.client.GetContainerLogs(id, 50)
		if err != nil {
			return logsMsg("(unable to fetch logs)")
		}
		return logsMsg(logs)
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

// ── Helpers ─────────────────────────────────────────────────────────────

func appendHist(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > historyLen {
		h = h[len(h)-historyLen:]
	}
	return h
}
