package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akib/docker-tui/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type view int

const (
	listView view = iota
	detailView
)

type Model struct {
	client       *docker.Client
	containers   []docker.ContainerInfo
	cursor       int
	currentView  view
	width        int
	height       int
	err          error
	notification string
	notifyIsErr  bool
	notifyTime   time.Time
	detailScroll int
	detailTab    int // 0=info, 1=env, 2=logs
	logs         string
	loading      bool
	inspected    *docker.ContainerInfo
}

// Messages
type containersMsg []docker.ContainerInfo
type errMsg struct{ err error }
type inspectMsg struct{ info *docker.ContainerInfo }
type logsMsg string
type tickMsg time.Time
type actionDoneMsg struct {
	action string
	name   string
}

type initMsg struct {
	client     *docker.Client
	containers []docker.ContainerInfo
}

func NewModel() Model {
	return Model{loading: true}
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
	return initMsg{client: c, containers: containers}
}

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
		m.loading = false
		return m, nil

	case containersMsg:
		m.containers = []docker.ContainerInfo(msg)
		m.loading = false
		if m.cursor >= len(m.containers) && m.cursor > 0 {
			m.cursor = len(m.containers) - 1
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
		if m.currentView == listView && m.client != nil {
			cmds = append(cmds, m.refreshContainers())
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if time.Since(m.notifyTime) > 4*time.Second {
			m.notification = ""
		}
		switch m.currentView {
		case listView:
			return m.updateListView(msg)
		case detailView:
			return m.updateDetailView(msg)
		}
	}
	return m, nil
}

func (m Model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m Model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "h", "backspace":
		m.currentView = listView
		m.inspected = nil
		m.logs = ""
		return m, m.refreshContainers()
	case "ctrl+c":
		if m.client != nil {
			m.client.Close()
		}
		return m, tea.Quit
	case "tab", "right":
		m.detailTab = (m.detailTab + 1) % 3
		m.detailScroll = 0
	case "shift+tab", "left":
		m.detailTab = (m.detailTab + 2) % 3
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
		return actionDoneMsg{action: "Started", name: name}
	}
}

func (m Model) stopContainer(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.StopContainer(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{action: "Stopped", name: name}
	}
}

// ── View ────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	var content string
	switch m.currentView {
	case listView:
		content = m.viewList()
	case detailView:
		content = m.viewDetail()
	}

	// Pad output to fill terminal height to prevent flickering
	lines := strings.Count(content, "\n") + 1
	if lines < m.height {
		content += strings.Repeat("\n", m.height-lines)
	}

	return content
}

// ── List View ───────────────────────────────────────────────────────────

func (m Model) viewList() string {
	var b strings.Builder
	w := m.width

	// ── Header bar (full width) ─────────────────────────────────────
	b.WriteString(m.renderHeader(w))

	// ── Error state ─────────────────────────────────────────────────
	if m.err != nil && len(m.containers) == 0 {
		boxW := min(w-4, 70)
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDanger).
			Foreground(colorDanger).
			Padding(1, 2).
			Width(boxW).
			Render(fmt.Sprintf("Cannot connect to Docker:\n\n%v\n\nMake sure Docker is running.", m.err))
		b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, errBox) + "\n\n")
		b.WriteString(m.renderHelpCentered(w))
		return b.String()
	}

	// ── Stat cards ──────────────────────────────────────────────────
	b.WriteString(m.renderStatCards(w) + "\n")

	// ── Empty state ─────────────────────────────────────────────────
	if len(m.containers) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Render("No containers found. Start some containers and press 'r' to refresh.")
		b.WriteString("\n" + lipgloss.PlaceHorizontal(w, lipgloss.Center, empty) + "\n\n")
		b.WriteString(m.renderHelpCentered(w))
		return b.String()
	}

	// ── Table ───────────────────────────────────────────────────────
	cols := m.calcColumns()
	b.WriteString(m.renderTableHeader(cols) + "\n")

	// Determine how many rows fit on screen
	// header(3) + stats(4) + table-header(2) + help(2) + notification(1) + padding(1)
	usedLines := 13
	visibleRows := m.height - usedLines
	if visibleRows < 3 {
		visibleRows = 3
	}

	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.containers))

	for i := startIdx; i < endIdx; i++ {
		b.WriteString(m.renderTableRow(i, cols) + "\n")
	}

	// Scroll indicator
	if len(m.containers) > visibleRows {
		pct := float64(m.cursor) / float64(max(len(m.containers)-1, 1)) * 100
		scroll := lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.cursor+1, len(m.containers), pct))
		b.WriteString(scroll + "\n")
	}

	// Notification
	b.WriteString(m.renderNotification())

	// Help bar
	b.WriteString("\n" + m.renderHelpCentered(w))

	return b.String()
}

func (m Model) renderHeader(w int) string {
	logo := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorPrimary).
		Render("  DOCKER TUI")

	// Right-aligned timestamp
	ts := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(time.Now().Format("15:04:05"))

	// Build the bar
	leftLen := lipgloss.Width(logo)
	rightLen := lipgloss.Width(ts)
	gap := w - leftLen - rightLen - 2
	if gap < 1 {
		gap = 1
	}

	bar := lipgloss.NewStyle().
		Background(colorBgAlt).
		Width(w).
		Render(logo + strings.Repeat(" ", gap) + ts + " ")

	// Accent line below header
	line := lipgloss.NewStyle().
		Foreground(colorDim).
		Render(strings.Repeat("─", w))

	return bar + "\n" + line + "\n"
}

func (m Model) renderStatCards(w int) string {
	running := 0
	stopped := 0
	other := 0
	for _, c := range m.containers {
		switch c.State {
		case "running":
			running++
		case "exited", "dead":
			stopped++
		default:
			other++
		}
	}
	total := len(m.containers)

	// Responsive card width
	cardW := 18
	if w >= 100 {
		cardW = 22
	} else if w < 60 {
		cardW = 14
	}

	makeCard := func(label, value string, valColor lipgloss.Color) string {
		v := statCardValue.Foreground(valColor).Width(cardW - 4).Render(value)
		l := statCardLabel.Width(cardW - 4).Render(label)
		return statCardBorder.Width(cardW).BorderForeground(valColor).Render(l + "\n" + v)
	}

	cards := []string{
		makeCard("TOTAL", fmt.Sprintf("%d", total), colorPrimary),
		makeCard("RUNNING", fmt.Sprintf("%d", running), colorSuccess),
		makeCard("STOPPED", fmt.Sprintf("%d", stopped), colorDanger),
	}

	if other > 0 {
		cards = append(cards, makeCard("OTHER", fmt.Sprintf("%d", other), colorWarning))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, interleave(cards, "  ")...)
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, row) + "\n"
}

// interleave inserts a spacer string between items for JoinHorizontal
func interleave(items []string, spacer string) []string {
	if len(items) == 0 {
		return items
	}
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			result = append(result, spacer)
		}
		result = append(result, item)
	}
	return result
}

// ── Responsive columns ──────────────────────────────────────────────────

type columns struct {
	state, name, image, status, ports, id int
	showPorts, showID                     bool
}

func (m Model) calcColumns() columns {
	w := m.width - 6 // margins
	c := columns{state: 3, showPorts: true, showID: true}

	if w < 50 {
		// Tiny terminal — just name + status
		c.showPorts = false
		c.showID = false
		c.name = max((w-c.state)*50/100, 10)
		c.image = max((w-c.state)*25/100, 8)
		c.status = w - c.state - c.name - c.image
	} else if w < 90 {
		// Medium — hide ID
		c.showID = false
		c.name = w * 25 / 100
		c.image = w * 30 / 100
		c.status = w * 22 / 100
		c.ports = w - c.state - c.name - c.image - c.status
	} else {
		// Wide — show everything
		c.name = w * 22 / 100
		c.image = w * 25 / 100
		c.status = w * 20 / 100
		c.ports = w * 18 / 100
		c.id = w - c.state - c.name - c.image - c.status - c.ports
	}

	return c
}

func (m Model) renderTableHeader(c columns) string {
	parts := fmt.Sprintf("  %s %s %s %s",
		tableHeaderStyle.Width(c.state).Render(""),
		tableHeaderStyle.Width(c.name).Render("NAME"),
		tableHeaderStyle.Width(c.image).Render("IMAGE"),
		tableHeaderStyle.Width(c.status).Render("STATUS"),
	)
	if c.showPorts {
		parts += " " + tableHeaderStyle.Width(c.ports).Render("PORTS")
	}
	if c.showID {
		parts += " " + tableHeaderStyle.Width(c.id).Render("ID")
	}
	return listHeaderStyle.Width(m.width).Render(parts)
}

func (m Model) renderTableRow(i int, c columns) string {
	ct := m.containers[i]
	icon := stateIcon(ct.State)
	stStyle := stateStyle(ct.State)

	row := fmt.Sprintf("%s %s %s %s",
		stStyle.Width(c.state).Render(icon),
		lipgloss.NewStyle().Width(c.name).Foreground(colorText).Render(truncate(ct.Name, c.name-1)),
		lipgloss.NewStyle().Width(c.image).Foreground(colorSubtext).Render(truncate(ct.Image, c.image-1)),
		lipgloss.NewStyle().Width(c.status).Foreground(colorSubtext).Render(truncate(ct.Status, c.status-1)),
	)
	if c.showPorts {
		p := formatPortsSummary(ct.Ports)
		row += " " + lipgloss.NewStyle().Width(c.ports).Foreground(colorSecondary).Render(truncate(p, c.ports-1))
	}
	if c.showID {
		row += " " + lipgloss.NewStyle().Width(c.id).Foreground(colorMuted).Render(ct.ID)
	}

	if i == m.cursor {
		return cursorStyle.Render("▸ ") + listItemSelectedStyle.Width(m.width-4).Render(row)
	}
	return "  " + listItemStyle.Width(m.width-4).Render(row)
}

// ── Detail View ─────────────────────────────────────────────────────────

func (m Model) viewDetail() string {
	if m.inspected == nil {
		if m.loading {
			return m.renderHeader(m.width) + "\n  Loading container details..."
		}
		return m.renderHeader(m.width) + "\n  No container selected."
	}

	var b strings.Builder
	c := m.inspected
	w := m.width

	// ── Header bar ──────────────────────────────────────────────────
	b.WriteString(m.renderHeader(w))

	// ── Container identity line ─────────────────────────────────────
	icon := stateIcon(c.State)
	stStyle := stateStyle(c.State)
	identity := "  " +
		lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(c.Name) + "  " +
		stStyle.Render(icon+" "+c.State) + "  " +
		lipgloss.NewStyle().Foreground(colorMuted).Render(c.ID)
	b.WriteString(identity + "\n\n")

	// ── Tabs ────────────────────────────────────────────────────────
	tabs := []string{"Info", "Environment", "Logs"}
	var tabRow strings.Builder
	tabRow.WriteString("  ")
	for i, t := range tabs {
		if i == m.detailTab {
			tabRow.WriteString(activeTabStyle.Render(" " + t + " "))
		} else {
			tabRow.WriteString(inactiveTabStyle.Render(" " + t + " "))
		}
		tabRow.WriteString(" ")
	}
	b.WriteString(tabRow.String() + "\n")

	// ── Tab content ─────────────────────────────────────────────────
	contentWidth := w - 8
	if contentWidth < 30 {
		contentWidth = 30
	}

	var tabContent string
	switch m.detailTab {
	case 0:
		tabContent = m.renderInfoTab(c, contentWidth)
	case 1:
		tabContent = m.renderEnvTab(c, contentWidth)
	case 2:
		tabContent = m.renderLogsTab(contentWidth)
	}

	// Apply scroll within available height
	lines := strings.Split(tabContent, "\n")
	// header(3) + identity(2) + tabs(1) + box-border(4) + help(2) + notification(1) + pad(1)
	boxPadding := 14
	availHeight := m.height - boxPadding
	if availHeight < 5 {
		availHeight = 5
	}

	maxScroll := max(0, len(lines)-availHeight)
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
	end := min(m.detailScroll+availHeight, len(lines))
	visibleContent := strings.Join(lines[m.detailScroll:end], "\n")

	// Scroll indicator in box title
	scrollHint := ""
	if len(lines) > availHeight {
		scrollHint = fmt.Sprintf(" (%d/%d)", m.detailScroll+1, maxScroll+1)
	}

	box := detailBoxStyle.Width(w - 4).Render(visibleContent)
	if scrollHint != "" {
		box += lipgloss.NewStyle().Foreground(colorMuted).Render(scrollHint)
	}
	b.WriteString(box + "\n")

	// Notification + help
	b.WriteString(m.renderNotification())
	b.WriteString(m.renderDetailHelp(w))

	return b.String()
}

// ── Tab renderers ───────────────────────────────────────────────────────

func (m Model) renderInfoTab(c *docker.ContainerInfo, width int) string {
	var b strings.Builder

	// Two-column layout for wide terminals
	if width > 70 {
		b.WriteString(m.renderInfoTwoCol(c, width))
	} else {
		b.WriteString(m.renderInfoSingleCol(c, width))
	}

	// Ports
	if len(c.Ports) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Ports") + "\n")
		for _, p := range c.Ports {
			val := fmt.Sprintf("%s:%s -> %s/%s", p.HostIP, p.HostPort, p.ContPort, p.Protocol)
			if p.HostIP == "" && p.HostPort == "" {
				val = fmt.Sprintf("%s/%s (not published)", p.ContPort, p.Protocol)
			}
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSecondary).Render("-> ") +
				lipgloss.NewStyle().Foreground(colorText).Render(val) + "\n")
		}
	}

	// Mounts
	if len(c.Mounts) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Mounts") + "\n")
		maxSrc := min(40, width/3)
		for _, mt := range c.Mounts {
			mode := "ro"
			if mt.RW {
				mode = "rw"
			}
			val := fmt.Sprintf("[%s] %s -> %s (%s)", mt.Type, truncate(mt.Source, maxSrc), mt.Destination, mode)
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorWarning).Render("* ") +
				lipgloss.NewStyle().Foreground(colorText).Render(val) + "\n")
		}
	}

	// Networks
	if len(c.Network) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Networks") + "\n")
		for name, net := range c.Network {
			b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorSecondary).Render(name) + "\n")
			if net.IPAddress != "" {
				b.WriteString(renderKV("  IP", net.IPAddress))
			}
			if net.Gateway != "" {
				b.WriteString(renderKV("  Gateway", net.Gateway))
			}
			if net.MacAddress != "" {
				b.WriteString(renderKV("  MAC", net.MacAddress))
			}
		}
	}

	// Labels
	if len(c.Labels) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Labels") + "\n")
		for k, v := range c.Labels {
			maxVal := max(width-len(k)-8, 10)
			label := truncate(k, 30) + "=" + truncate(v, maxVal)
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).Render(label) + "\n")
		}
	}

	return b.String()
}

func (m Model) renderInfoTwoCol(c *docker.ContainerInfo, width int) string {
	halfW := (width - 4) / 2

	var left, right strings.Builder

	left.WriteString(sectionHeaderStyle.Width(halfW).Render("  Identity") + "\n")
	left.WriteString(renderKV("Name", c.Name))
	left.WriteString(renderKV("ID", c.ID))
	left.WriteString(renderKV("Image", truncate(c.Image, halfW-18)))
	left.WriteString(renderKV("Command", truncate(c.Command, halfW-18)))

	right.WriteString(sectionHeaderStyle.Width(halfW).Render("  Runtime") + "\n")
	right.WriteString(renderKV("Status", c.Status))
	right.WriteString(renderKV("Created", c.Created.Format("2006-01-02 15:04")))
	if c.Platform != "" {
		right.WriteString(renderKV("Platform", c.Platform))
	}
	if c.RestartCount > 0 {
		right.WriteString(renderKV("Restarts", fmt.Sprintf("%d", c.RestartCount)))
	}

	leftBox := lipgloss.NewStyle().Width(halfW).Render(left.String())
	rightBox := lipgloss.NewStyle().Width(halfW).Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, "  ", rightBox) + "\n"
}

func (m Model) renderInfoSingleCol(c *docker.ContainerInfo, width int) string {
	var b strings.Builder
	b.WriteString(sectionHeaderStyle.Width(width).Render("  General") + "\n")
	b.WriteString(renderKV("ID", c.ID))
	b.WriteString(renderKV("Name", c.Name))
	b.WriteString(renderKV("Image", c.Image))
	b.WriteString(renderKV("Command", c.Command))
	b.WriteString(renderKV("Created", c.Created.Format("2006-01-02 15:04:05")))
	b.WriteString(renderKV("Status", c.Status))
	if c.Platform != "" {
		b.WriteString(renderKV("Platform", c.Platform))
	}
	if c.RestartCount > 0 {
		b.WriteString(renderKV("Restarts", fmt.Sprintf("%d", c.RestartCount)))
	}
	return b.String()
}

func (m Model) renderEnvTab(c *docker.ContainerInfo, width int) string {
	if len(c.Env) == 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No environment variables available.\n  (Container must be inspected with sufficient permissions)")
	}

	var b strings.Builder
	b.WriteString(sectionHeaderStyle.Width(width).Render(
		fmt.Sprintf("  Environment Variables (%d)", len(c.Env))) + "\n")

	for _, env := range c.Env {
		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		val := ""
		if len(parts) > 1 {
			val = parts[1]
		}
		maxVal := max(width-len(key)-6, 10)
		keyStyled := lipgloss.NewStyle().Foreground(colorSecondary).Bold(true).Render(key)
		valStyled := lipgloss.NewStyle().Foreground(colorText).Render("=" + truncate(val, maxVal))
		b.WriteString("  " + keyStyled + valStyled + "\n")
	}
	return b.String()
}

func (m Model) renderLogsTab(width int) string {
	if m.logs == "" {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No logs available.")
	}

	var b strings.Builder
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Container Logs (last 50 lines)") + "\n")

	cleaned := cleanDockerLogs(m.logs)
	for _, line := range strings.Split(cleaned, "\n") {
		if len(line) > width-4 {
			line = line[:width-4]
		}
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).Render(line) + "\n")
	}
	return b.String()
}

// ── Help bars ───────────────────────────────────────────────────────────

func (m Model) renderHelpCentered(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"enter", "details"},
		{"s", "start/stop"},
		{"r", "refresh"},
		{"q", "quit"},
	}
	return helpBarStyle.Render(lipgloss.PlaceHorizontal(w, lipgloss.Center, formatHelpKeys(keys)))
}

func (m Model) renderDetailHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"</>", "tabs"},
		{"j/k", "scroll"},
		{"s", "start/stop"},
		{"esc", "back"},
	}
	return helpBarStyle.Render(lipgloss.PlaceHorizontal(w, lipgloss.Center, formatHelpKeys(keys)))
}

func formatHelpKeys(keys []struct{ key, desc string }) string {
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.key)+" "+helpDescStyle.Render(k.desc))
	}
	return strings.Join(parts, "  "+lipgloss.NewStyle().Foreground(colorDim).Render("|")+"  ")
}

// ── Notification ────────────────────────────────────────────────────────

func (m Model) renderNotification() string {
	if m.notification == "" || time.Since(m.notifyTime) > 4*time.Second {
		return ""
	}
	if m.notifyIsErr {
		return notifyErrorStyle.Render("  "+m.notification) + "\n"
	}
	return notifySuccessStyle.Render("  "+m.notification) + "\n"
}

// ── Helpers ─────────────────────────────────────────────────────────────

func renderKV(key, value string) string {
	return "  " + detailLabelStyle.Render(key) + " " + detailValueStyle.Render(value) + "\n"
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func formatPortsSummary(ports []docker.PortBinding) string {
	if len(ports) == 0 {
		return "-"
	}
	var parts []string
	seen := make(map[string]bool)
	for _, p := range ports {
		var s string
		if p.HostPort != "" && p.HostPort != "0" {
			s = p.HostPort + "->" + p.ContPort
		} else {
			s = p.ContPort
		}
		if !seen[s] {
			parts = append(parts, s)
			seen[s] = true
		}
	}
	return strings.Join(parts, ",")
}

func cleanDockerLogs(s string) string {
	var cleaned strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if len(line) > 8 {
			header := line[0]
			if header == 1 || header == 2 {
				line = line[8:]
			}
		}
		cleaned.WriteString(line + "\n")
	}
	return strings.TrimRight(cleaned.String(), "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
