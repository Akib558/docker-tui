package ui

import (
	"io"
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
)

// ── View / Dialog enums ─────────────────────────────────────────────────

type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewImages
	viewEvents
	viewLogs
	viewVolumes
	viewNetworks
	viewNotifications
)

type dialogMode int

const (
	dialogNone dialogMode = iota
	dialogConfirm
	dialogTheme
	dialogInput
	dialogHelp
	dialogCommandPalette
)

// ── Detail tab indices ───────────────────────────────────────────────────

const (
	tabInfo      = 0
	tabResources = 1
	tabEnv       = 2
	tabLogs      = 3
	tabTerminal  = 4
	tabDiff      = 5
	tabProcesses = 6
	tabCount     = 7
)

const historyLen = 60
const terminalBufferMax = 96 * 1024

// ── Model ───────────────────────────────────────────────────────────────

type Model struct {
	// Docker
	client            docker.ClientAPI
	containers        []docker.ContainerInfo
	inspected         *docker.ContainerInfo
	images            []docker.ImageInfo
	imagePullProgress string
	networks          []docker.NetworkResource
	events            []docker.DockerEvent
	volumes           []docker.VolumeInfo
	volCursor         int
	netCursor         int

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
	view         viewState
	cursor       int
	imgCursor    int
	eventsCursor int
	width        int
	height       int

	// Caches (Phase 3)
	filteredCache    []docker.ContainerInfo
	filteredCacheKey filteredCacheKey
	dashboardCache   string
	dashboardCacheW  int
	containerNames   map[string]string

	// Detail
	detailScroll         int
	detailTab            int
	logViewer            LogViewerState
	logCancel            func()
	liveLogging          bool
	diff                 []docker.DiffEntry
	processTop           docker.ContainerTop
	processLoaded        bool
	terminalInput        string
	terminalOutput       string
	terminalFollow       bool
	terminalCancel       func()
	terminalWriter       io.Writer
	terminalActive       bool
	terminalShell        string
	terminalInputFocused bool

	// Centralized logs
	centralLogs         LogViewerState
	centralLogTargets   []LogTarget
	centralLogCancels   []func()
	centralLogFiltering bool
	centralLogFilter    string
	centralLogRegex     bool

	// Events streaming
	eventsCancel func()

	// Filter
	filtering     bool
	filterText    string
	volFiltering  bool
	volFilterText string
	netFiltering  bool
	netFilterText string

	// Multi-select
	selected map[string]bool

	// Dialog
	dialog      dialogMode
	helpScroll  int
	confirmMsg  string
	confirmOK   tea.Cmd
	inputText   string
	inputPrompt string
	inputSubmit func(string) tea.Cmd

	// Command palette
	commandPaletteText    string
	commandPaletteCursor  int
	commandPaletteResults []Command

	// Theme
	themeCursor int

	// Notification
	notification  string
	notifyIsErr   bool
	notifyTime    time.Time
	notifyHistory []Notification
	notifyCursor  int

	// Config and state
	cfg             *config.Config
	refreshInterval time.Duration
	loading         bool
	lastRefresh     time.Time
	err             error
	startTime       time.Time
	tickCount       int
	groupByCompose  bool
	sortMode        int

	// Reconnection
	reconnecting      bool
	reconnectAttempts int
}

type filteredCacheKey struct {
	filter   string
	sortMode int
	n        int
}

type Command struct {
	Name        string
	Description string
	Run         func(Model) (Model, tea.Cmd)
}

type Notification struct {
	Message   string
	IsError   bool
	Timestamp time.Time
}

// Sort modes
const (
	sortName = iota
	sortState
	sortCPU
	sortMemory
	sortImage
	sortModeCount
)

func statePriority(state string) int {
	switch state {
	case "running":
		return 0
	case "restarting":
		return 1
	case "paused":
		return 2
	case "created":
		return 3
	case "exited":
		return 4
	case "dead":
		return 5
	default:
		return 6
	}
}

// ── Messages ────────────────────────────────────────────────────────────

type containersMsg []docker.ContainerInfo
type imagesMsg []docker.ImageInfo
type errMsg struct{ err error }
type inspectMsg struct{ info *docker.ContainerInfo }
type logsMsg []LogEntry
type tickMsg time.Time
type actionDoneMsg struct{ action, name string }
type statsMsg struct {
	stats   map[string]*docker.ContainerResourceStats
	sysMem  docker.SystemMemory
	sysLoad docker.SystemLoad
}
type diffMsg []docker.DiffEntry
type topMsg struct{ top docker.ContainerTop }
type imageActionDoneMsg struct{ action, name string }
type pullProgressMsg struct {
	text string
	next tea.Cmd
}
type volumesMsg []docker.VolumeInfo
type volumeActionDoneMsg struct{ action, name string }
type networksMsg []docker.NetworkResource
type networkActionDoneMsg struct{ action, name string }
type execDoneMsg struct{ err error }
type loadHistMsg struct {
	cpu map[string][]float64
	mem map[string][]float64
}
type logLineMsg struct {
	entry LogEntry
	next  tea.Cmd
}
type logStreamStartMsg struct {
	cancel func()
	next   tea.Cmd
}
type logStreamDoneMsg struct{}
type centralLogTailMsg struct {
	entries []LogEntry
}
type centralLogLineMsg struct {
	entry LogEntry
	next  tea.Cmd
}
type centralLogStreamStartMsg struct {
	cancel func()
	next   tea.Cmd
}
type centralLogStreamDoneMsg struct {
	target LogTarget
	err    error
}
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

type reconnectMsg struct {
	success bool
	err     error
	client  docker.ClientAPI
}

type initMsg struct {
	client     docker.ClientAPI
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
		refreshInterval: cfg.RefreshDuration(),
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
