package ui

import (
	"strings"
	"sync"

	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

type headerMemoKey struct {
	w              int
	filtering      bool
	volFiltering   bool
	netFiltering   bool
	view           viewState
	filterText     string
	volFilterText  string
	netFilterText  string
	selectedN      int
	reconnecting   bool
	overviewVer    string
	overviewImages int
	sortMode       int
	clockSec       string
}

type detailChromeKey struct {
	id    string
	w     int
	tab   int
	state string
	image string
}

var (
	frameMemoMu sync.Mutex
	frameMemo   struct {
		headerKey headerMemoKey
		header    string
		dashKey   dashboardMemoKey
		dash      string
		detailKey detailChromeKey
		detail    string
	}
)

type dashboardMemoKey struct {
	w           int
	nContainers int
	statsGen    int
	widthTier   int // 0 narrow, 1 wide
	summarySig  uint64
	memTotal    uint64
	load1       uint64
	histLen     int
	refreshMeta string
}

func (m Model) headerMemoKey(w int) headerMemoKey {
	k := headerMemoKey{
		w:             w,
		filtering:     m.filtering,
		volFiltering:  m.volFiltering,
		netFiltering:  m.netFiltering,
		view:          m.view,
		filterText:    m.filterText,
		volFilterText: m.volFilterText,
		netFilterText: m.netFilterText,
		selectedN:     len(m.selected),
		reconnecting:  m.reconnecting,
		sortMode:      m.sortMode,
		clockSec:      nowFunc().Format("15:04:05"),
	}
	if m.overview != nil {
		k.overviewVer = m.overview.ServerVersion
		k.overviewImages = m.overview.Images
	}
	return k
}

func (m Model) memoizedHeader(w int) string {
	k := m.headerMemoKey(w)
	frameMemoMu.Lock()
	defer frameMemoMu.Unlock()
	if frameMemo.headerKey == k && frameMemo.header != "" {
		return frameMemo.header
	}
	frameMemo.header = m.buildHeader(w)
	frameMemo.headerKey = k
	return frameMemo.header
}

func (m Model) dashboardMemoKey(w int) dashboardMemoKey {
	tier := 0
	if w >= 80 {
		tier = 1
	}
	k := dashboardMemoKey{
		w:           w,
		nContainers: len(m.containers),
		statsGen:    m.statsGen,
		widthTier:   tier,
		refreshMeta: m.renderRefreshMeta(),
	}
	for _, seg := range m.containerStateSummary() {
		k.summarySig = k.summarySig*31 + uint64(seg.count)
	}
	if m.systemMem.Total > 0 {
		k.memTotal = m.systemMem.Total
	}
	if m.systemLoad.Load1 > 0 {
		k.load1 = uint64(m.systemLoad.Load1 * 1000)
	}
	if h := m.hostCPUHistory(); len(h) > 0 {
		k.histLen = len(h)
	}
	return k
}

func (m Model) memoizedStatusStrip(w int) string {
	if len(m.containers) == 0 {
		return ""
	}
	k := m.dashboardMemoKey(w)
	frameMemoMu.Lock()
	defer frameMemoMu.Unlock()
	if frameMemo.dashKey == k && frameMemo.dash != "" {
		return frameMemo.dash
	}
	frameMemo.dash = m.renderStatusStrip(w)
	frameMemo.dashKey = k
	return frameMemo.dash
}

func invalidateFrameMemo() {
	frameMemoMu.Lock()
	frameMemo.header = ""
	frameMemo.dash = ""
	frameMemo.detail = ""
	frameMemoMu.Unlock()
}

func (m Model) memoizedDetailChrome(c *docker.ContainerInfo, w int) string {
	if c == nil {
		return ""
	}
	k := detailChromeKey{id: c.ID, w: w, tab: m.detailTab, state: c.State, image: c.Image}
	frameMemoMu.Lock()
	defer frameMemoMu.Unlock()
	if frameMemo.detailKey == k && frameMemo.detail != "" {
		return frameMemo.detail
	}
	boxWidth := max(w-4, 30)
	var b strings.Builder
	b.WriteString(m.renderDetailIdentity(c, w))
	b.WriteString("\n\n")
	bar := clampRenderedWidth(m.renderDetailTabs(boxWidth), w)
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", boxWidth))
	b.WriteString(bar + "\n" + sep + "\n")
	frameMemo.detail = b.String()
	frameMemo.detailKey = k
	return frameMemo.detail
}
