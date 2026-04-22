package ui

import (
	"io"
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

// ── Detail tab indices ───────────────────────────────────────────────────

const (
	tabInfo      = 0
	tabResources = 1
	tabEnv       = 2
	tabLogs      = 3
	tabTerminal  = 4
	tabCount     = 5
)

const historyLen = 60
const terminalBufferMax = 96 * 1024

// ── Model ───────────────────────────────────────────────────────────────

type Model struct {
	// Docker
	client     *docker.Client
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
	view      viewState
	cursor    int
	imgCursor int
	width     int
	height    int

	// Detail
	detailScroll   int
	detailTab      int
	logLines       []string
	logCancel      func()
	liveLogging    bool
	diff           []docker.DiffEntry
	terminalInput  string
	terminalOutput string
	terminalFollow bool
	terminalCancel func()
	terminalWriter io.Writer
	terminalActive bool
	terminalShell  string

	// Events streaming
	eventsCtx    interface{} // unused field kept for future use
	eventsCancel func()

	// Filter
	filtering  bool
	filterText string

	// Multi-select
	selected map[string]bool

	// Dialog
	dialog      dialogMode
	confirmMsg  string
	confirmOK   tea.Cmd
	inputText   string
	inputPrompt string
	inputSubmit func(string) tea.Cmd

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
	tickCount   int

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
