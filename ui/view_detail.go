package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

// ═══════════════════════════════════════════════════════════════════════
//  DETAIL VIEW
// ═══════════════════════════════════════════════════════════════════════

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
	boxWidth := max(w-4, 30)
	contentWidth := max(boxWidth-6, 24)

	b.WriteString(m.renderHeader(w))
	b.WriteString(m.renderDetailIdentity(c, w) + "\n\n")

	tabLine := m.renderDetailTabs(boxWidth)
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", boxWidth))
	b.WriteString(tabLine + "\n" + sep + "\n")

	var tabContent string
	switch m.detailTab {
	case tabInfo:
		tabContent = m.renderInfoTab(c, contentWidth)
	case tabResources:
		tabContent = m.renderResourcesTab(c, contentWidth)
	case tabEnv:
		tabContent = m.renderEnvTab(c, contentWidth)
	case tabLogs:
		tabContent = m.renderLogsTab(contentWidth)
	case tabTerminal:
		tabContent = m.renderTerminalTab(c, contentWidth)
	case tabDiff:
		tabContent = m.renderDiffTab(c, contentWidth)
	case tabProcesses:
		tabContent = m.renderProcessesTab(c, contentWidth)
	}

	lines := strings.Split(tabContent, "\n")
	availHeight := m.detailBoxInnerHeight()
	maxScroll := max(0, len(lines)-availHeight)
	scrollPos := m.detailScroll
	followMode := m.terminalFollow
	switch m.detailTab {
	case tabLogs:
		scrollPos = 0 // log viewer handles its own scroll window
	case tabTerminal:
		scrollPos, followMode = normalizeTerminalScroll(m.detailScroll, maxScroll, m.terminalFollow)
	default:
		if scrollPos > maxScroll {
			scrollPos = maxScroll
		}
	}
	if scrollPos < 0 {
		scrollPos = 0
	}
	end := min(scrollPos+availHeight, len(lines))
	visible := strings.Join(lines[scrollPos:end], "\n")

	box := detailBoxStyle.Width(boxWidth).Render(visible)
	if len(lines) > availHeight && m.detailTab != tabLogs {
		scrollLabel := fmt.Sprintf(" (%d/%d)", scrollPos+1, maxScroll+1)
		if m.detailTab == tabTerminal {
			mode := "PAUSED"
			if followMode {
				mode = "FOLLOW"
			}
			scrollLabel = fmt.Sprintf(" [%s] %d/%d", mode, scrollPos+1, maxScroll+1)
		}
		box += lipgloss.NewStyle().Foreground(colorMuted).Render(scrollLabel)
	}
	b.WriteString(box + "\n")

	b.WriteString(m.renderNotification())
	b.WriteString(m.detailHelp(w))
	return b.String()
}

func (m Model) renderDetailTabs(width int) string {
	tabNames := []string{"Info", "Resources", "Environment", "Logs", "Terminal", "Diff", "Processes"}
	parts := make([]string, 0, len(tabNames))
	for i, t := range tabNames {
		num := lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("%d:", i+1))
		if i == m.detailTab {
			parts = append(parts, activeTabStyle.Render(t))
		} else {
			parts = append(parts, num+" "+inactiveTabStyle.Render(t))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, interleave(parts, "  ")...)
	return lipgloss.NewStyle().Width(width).Render(row)
}

// ── Info tab ────────────────────────────────────────────────────────────

func (m Model) renderInfoTab(c *docker.ContainerInfo, width int) string {
	var b strings.Builder

	if width > 70 {
		b.WriteString(m.renderInfoTwoCol(c, width))
	} else {
		b.WriteString(m.renderInfoSingleCol(c, width))
	}

	// Resource limits section
	if c.CPUQuota > 0 || c.MemoryLimit > 0 || c.RestartPolicy != "" {
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Resource Limits") + "\n")
		if c.CPUQuota > 0 {
			cpus := float64(c.CPUQuota) / float64(c.CPUPeriod)
			b.WriteString(renderKV("CPU Quota", fmt.Sprintf("%.2f cores", cpus)))
		}
		if c.MemoryLimit > 0 {
			b.WriteString(renderKV("Memory Limit", formatBytes(uint64(c.MemoryLimit))))
		}
		if c.RestartPolicy != "" {
			b.WriteString(renderKV("Restart Policy", c.RestartPolicy))
		}
	}

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

	if len(c.Labels) > 0 {
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Labels") + "\n")
		for k, v := range c.Labels {
			maxVal := max(width-len(k)-8, 10)
			b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).
				Render(truncate(k, 30)+"="+truncate(v, maxVal)) + "\n")
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
	if c.Health != "" {
		right.WriteString(renderHealthKV("Health", c.Health))
	}
	if !c.StartedAt.IsZero() && c.State == "running" {
		right.WriteString(renderKV("Uptime", formatUptime(time.Since(c.StartedAt))))
	}
	if !c.Created.IsZero() {
		right.WriteString(renderKV("Created", c.Created.Format("2006-01-02 15:04")))
	}
	if c.Platform != "" {
		right.WriteString(renderKV("Platform", c.Platform))
	}
	if c.RestartCount > 0 {
		right.WriteString(renderRestartKV("Restarts", c.RestartCount))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(halfW).Render(left.String()),
		"  ",
		lipgloss.NewStyle().Width(halfW).Render(right.String()),
	) + "\n"
}

func (m Model) renderInfoSingleCol(c *docker.ContainerInfo, width int) string {
	var b strings.Builder
	b.WriteString(sectionHeaderStyle.Width(width).Render("  General") + "\n")
	b.WriteString(renderKV("ID", c.ID))
	b.WriteString(renderKV("Name", c.Name))
	b.WriteString(renderKV("Image", c.Image))
	b.WriteString(renderKV("Command", c.Command))
	if !c.Created.IsZero() {
		b.WriteString(renderKV("Created", c.Created.Format("2006-01-02 15:04:05")))
	}
	b.WriteString(renderKV("Status", c.Status))
	if c.Health != "" {
		b.WriteString(renderHealthKV("Health", c.Health))
	}
	if !c.StartedAt.IsZero() && c.State == "running" {
		b.WriteString(renderKV("Uptime", formatUptime(time.Since(c.StartedAt))))
	}
	if c.Platform != "" {
		b.WriteString(renderKV("Platform", c.Platform))
	}
	if c.RestartCount > 0 {
		b.WriteString(renderRestartKV("Restarts", c.RestartCount))
	}
	return b.String()
}

// ── Resources tab ───────────────────────────────────────────────────────

func (m Model) renderResourcesTab(c *docker.ContainerInfo, width int) string {
	s, hasStats := m.stats[c.ID]
	if !hasStats || c.State != "running" {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No resource data available.\n  (Container must be running)")
	}

	cpuHist := m.cpuHistory[c.ID]
	memHist := m.memHistory[c.ID]

	if width > 70 {
		return m.renderResourcesTwoCol(s, cpuHist, memHist, width)
	}
	return m.renderResourcesSingleCol(s, cpuHist, memHist, width)
}

func (m Model) renderResourcesTwoCol(s *docker.ContainerResourceStats, cpuH, memH []float64, width int) string {
	halfW := (width - 4) / 2
	sparkW := halfW - 4
	barW := halfW - 10

	var left, right strings.Builder

	left.WriteString(sectionHeaderStyle.Width(halfW).Render("  CPU Usage") + "\n")
	left.WriteString("  " + sparklineColored(cpuH, sparkW, 100, colorPrimary) + "\n")
	left.WriteString("  " + renderBarSegments(s.CPUPercent, barW+5, "") + "\n")

	right.WriteString(sectionHeaderStyle.Width(halfW).Render("  Memory Usage") + "\n")
	right.WriteString("  " + sparklineColored(memH, sparkW, 100, colorCyan) + "\n")
	right.WriteString("  " + renderBarSegments(s.MemPercent, barW+5, "") + "\n")
	right.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).
		Render(fmt.Sprintf("%s / %s", formatBytes(s.MemUsage), formatBytes(s.MemLimit))) + "\n")

	out := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(halfW).Render(left.String()),
		"  ",
		lipgloss.NewStyle().Width(halfW).Render(right.String()),
	)
	out += "\n" + m.renderIOStats(s, width)
	return out
}

func (m Model) renderResourcesSingleCol(s *docker.ContainerResourceStats, cpuH, memH []float64, width int) string {
	var b strings.Builder
	sparkW := width - 4
	barW := width - 12

	b.WriteString(sectionHeaderStyle.Width(width).Render("  CPU Usage") + "\n")
	b.WriteString("  " + sparklineColored(cpuH, sparkW, 100, colorPrimary) + "\n")
	b.WriteString("  " + renderBarSegments(s.CPUPercent, barW+5, "") + "\n\n")

	b.WriteString(sectionHeaderStyle.Width(width).Render("  Memory Usage") + "\n")
	b.WriteString("  " + sparklineColored(memH, sparkW, 100, colorCyan) + "\n")
	b.WriteString("  " + renderBarSegments(s.MemPercent, barW+5, "") + "\n")
	b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).
		Render(fmt.Sprintf("%s / %s", formatBytes(s.MemUsage), formatBytes(s.MemLimit))) + "\n")
	b.WriteString("\n" + m.renderIOStats(s, width))
	return b.String()
}

func (m Model) renderIOStats(s *docker.ContainerResourceStats, width int) string {
	var b strings.Builder

	if width > 70 {
		halfW := (width - 4) / 2

		var left, right strings.Builder
		left.WriteString(sectionHeaderStyle.Width(halfW).Render("  Network I/O") + "\n")
		left.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).Render("↓ RX  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.NetRx)) + "\n")
		left.WriteString("  " + lipgloss.NewStyle().Foreground(colorDanger).Render("↑ TX  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.NetTx)) + "\n")

		right.WriteString(sectionHeaderStyle.Width(halfW).Render("  Block I/O") + "\n")
		right.WriteString("  " + lipgloss.NewStyle().Foreground(colorSecondary).Render("↓ Read   ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.BlockRead)) + "\n")
		right.WriteString("  " + lipgloss.NewStyle().Foreground(colorWarning).Render("↑ Write  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.BlockWrite)) + "\n")

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(halfW).Render(left.String()),
			"  ",
			lipgloss.NewStyle().Width(halfW).Render(right.String()),
		))
	} else {
		b.WriteString(sectionHeaderStyle.Width(width).Render("  Network I/O") + "\n")
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).Render("↓ RX  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.NetRx)) + "\n")
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorDanger).Render("↑ TX  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.NetTx)) + "\n")
		b.WriteString("\n" + sectionHeaderStyle.Width(width).Render("  Block I/O") + "\n")
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSecondary).Render("↓ Read   ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.BlockRead)) + "\n")
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorWarning).Render("↑ Write  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(formatBytes(s.BlockWrite)) + "\n")
	}

	if s.PIDs > 0 {
		b.WriteString("\n  " + lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("PIDs  ") +
			lipgloss.NewStyle().Foreground(colorText).Render(fmt.Sprintf("%d", s.PIDs)) + "\n")
	}
	return b.String()
}

// ── Environment tab ─────────────────────────────────────────────────────

func (m Model) renderEnvTab(c *docker.ContainerInfo, width int) string {
	if len(c.Env) == 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No environment variables available.")
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
		b.WriteString("  " +
			lipgloss.NewStyle().Foreground(colorSecondary).Bold(true).Render(key) +
			lipgloss.NewStyle().Foreground(colorText).Render("="+truncate(val, maxVal)) + "\n")
	}
	return b.String()
}

// ── Logs tab ────────────────────────────────────────────────────────────

func (m Model) renderLogsTab(width int) string {
	var b strings.Builder
	liveIndicator := ""
	if m.liveLogging {
		liveIndicator = " " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("● LIVE")
	}
	mode := "FOLLOW"
	if !m.logViewer.Follow {
		mode = "PAUSED"
	}
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Container Logs "+mode+liveIndicator) + "\n")
	if m.logViewer.ShowLegend {
		b.WriteString(renderLogLegend(width) + "\n")
	}
	rows := m.logViewer.VisibleEntries(m.detailLogContentRows())
	if len(rows) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("  No logs available.") + "\n")
		return b.String()
	}
	start, _, _ := m.logViewer.VisibleWindow(m.detailLogContentRows())
	for i, entry := range rows {
		b.WriteString(m.renderLogEntryRow(entry, width, false, start+i, i) + "\n")
	}
	return b.String()
}

// ── Terminal tab ────────────────────────────────────────────────────────

func (m Model) renderTerminalTab(c *docker.ContainerInfo, width int) string {
	if c.State != "running" {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Terminal is only available for running containers.")
	}

	var b strings.Builder
	state := "disconnected"
	if m.terminalActive {
		state = "connected"
	}
	header := fmt.Sprintf("  Embedded Terminal (%s)", state)
	b.WriteString(sectionHeaderStyle.Width(width).Render(header) + "\n")
	if m.terminalShell != "" {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).Render("shell: "+m.terminalShell) + "\n")
	}
	b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).
		Render("Ctrl+\\ detach · type to send input · Esc release focus") + "\n")
	if m.terminalInputFocused {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).
			Render("INPUT FOCUSED · Esc to release") + "\n")
	} else if m.terminalActive {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).
			Render("start typing or press i/Enter to focus input") + "\n")
	} else {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).
			Render("press x to connect") + "\n")
	}
	b.WriteString("\n")

	out := sanitizeOutputText(m.terminalOutput)
	if out == "" {
		out = "(no terminal output yet)"
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if lipgloss.Width(line) > width-4 {
			line = truncate(line, width-4)
		}
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorText).Render(line) + "\n")
	}
	prompt := "  > " + m.terminalInput
	b.WriteString("\n" + inputStyle.Width(width-2).Render(prompt))
	return b.String()
}

// ── Diff tab ────────────────────────────────────────────────────────────

func (m Model) renderDiffTab(c *docker.ContainerInfo, width int) string {
	var b strings.Builder
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Filesystem Changes") + "\n")

	if m.diff == nil {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("Press 'f' to load filesystem changes") + "\n")
		return b.String()
	}

	if len(m.diff) == 0 {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).
			Render("No filesystem changes detected.") + "\n")
		return b.String()
	}

	added, modified, deleted := 0, 0, 0
	for _, d := range m.diff {
		switch d.Kind {
		case "A":
			added++
		case "M":
			modified++
		case "D":
			deleted++
		}
	}

	summary := fmt.Sprintf("  %d added  %d modified  %d deleted", added, modified, deleted)
	b.WriteString(lipgloss.NewStyle().Foreground(colorSubtext).Render(summary) + "\n\n")

	for _, d := range m.diff {
		var icon string
		var style lipgloss.Style
		switch d.Kind {
		case "A":
			icon = "+"
			style = lipgloss.NewStyle().Foreground(colorSuccess)
		case "D":
			icon = "-"
			style = lipgloss.NewStyle().Foreground(colorDanger)
		default:
			icon = "~"
			style = lipgloss.NewStyle().Foreground(colorWarning)
		}
		path := d.Path
		if lipgloss.Width(path) > width-6 {
			path = truncate(path, width-6)
		}
		b.WriteString("  " + style.Bold(true).Render(icon+" ") + style.Render(path) + "\n")
	}
	return b.String()
}

func (m Model) renderProcessesTab(c *docker.ContainerInfo, width int) string {
	var b strings.Builder
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Running Processes") + "\n")

	if c.State != "running" && c.State != "paused" {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("Processes are available for running or paused containers.") + "\n")
		return b.String()
	}

	if !m.processLoaded {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("Loading process list...") + "\n")
		return b.String()
	}

	if len(m.processTop.Titles) == 0 {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("No process information available. Press 'p' to refresh.") + "\n")
		return b.String()
	}

	header := strings.Join(m.processTop.Titles, "  ")
	b.WriteString("  " + tableHeaderStyle.Render(truncate(header, max(width-4, 10))) + "\n")
	b.WriteString("  " + lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("-", max(width-4, 10))) + "\n")

	if len(m.processTop.Processes) == 0 {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render("No processes found.") + "\n")
		return b.String()
	}

	for _, row := range m.processTop.Processes {
		line := strings.Join(row, "  ")
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorText).Render(truncate(line, max(width-4, 10))) + "\n")
	}
	b.WriteString("\n  " + lipgloss.NewStyle().Foreground(colorMuted).Render("Press 'p' to refresh process list."))
	return b.String()
}

func (m Model) renderDetailIdentity(c *docker.ContainerInfo, w int) string {
	dot := lipgloss.NewStyle().Foreground(colorDim).Render("  ·  ")
	stateStr := StateBadgeStyled(c.State)
	reserve := 2 + lipgloss.Width(stateStr) + lipgloss.Width(dot)*3 + 16
	remain := max(w-reserve, 20)
	nameW := max(remain*55/100, 8)
	imgW := max(remain-nameW, 8)

	nameStr := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(truncateDisplay(c.Name, nameW))
	imgStr := lipgloss.NewStyle().Foreground(colorSubtext).Render(truncateDisplay(c.Image, imgW))
	idStr := lipgloss.NewStyle().Foreground(colorDim).Render(truncateDisplay(c.ID, 16))
	return "  " + nameStr + dot + stateStr + dot + imgStr + dot + idStr
}
