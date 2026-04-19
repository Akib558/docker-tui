package ui

import (
	"fmt"
	"strings"

	"github.com/akib/docker-tui/docker"
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

	b.WriteString(m.renderHeader(w))

	icon := stateIcon(c.State)
	stStyle := stateStyle(c.State)
	nameStr := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(c.Name)
	stateStr := stStyle.Render(icon + " " + c.State)
	idStr := lipgloss.NewStyle().Foreground(colorDim).Render(truncate(c.ID, 16))
	imgStr := lipgloss.NewStyle().Foreground(colorSubtext).Render(truncate(c.Image, max(w/3, 20)))
	dot := lipgloss.NewStyle().Foreground(colorDim).Render("  ·  ")
	identity := "  " + nameStr + dot + stateStr + dot + imgStr + dot + idStr
	b.WriteString(identity + "\n\n")

	tabNames := []string{"Info", "Resources", "Environment", "Logs", "Terminal"}
	var tabRow strings.Builder
	tabRow.WriteString("  ")
	for i, t := range tabNames {
		num := lipgloss.NewStyle().Foreground(colorDim).Render(fmt.Sprintf("%d:", i+1))
		label := num + " " + t
		if i == m.detailTab {
			tabRow.WriteString(activeTabStyle.Render(label))
		} else {
			tabRow.WriteString(inactiveTabStyle.Render(label))
		}
		tabRow.WriteString("  ")
	}
	tabLine := tabRow.String()
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", w))
	b.WriteString(tabLine + "\n" + sep + "\n")

	contentWidth := w - 8
	if contentWidth < 30 {
		contentWidth = 30
	}

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
	}

	lines := strings.Split(tabContent, "\n")
	boxChrome := 15
	availHeight := m.height - boxChrome
	if availHeight < 5 {
		availHeight = 5
	}
	maxScroll := max(0, len(lines)-availHeight)
	if m.detailScroll > maxScroll {
		m.detailScroll = maxScroll
	}
	end := min(m.detailScroll+availHeight, len(lines))
	visible := strings.Join(lines[m.detailScroll:end], "\n")

	box := detailBoxStyle.Width(w - 4).Render(visible)
	if len(lines) > availHeight {
		box += lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf(" (%d/%d)", m.detailScroll+1, maxScroll+1))
	}
	b.WriteString(box + "\n")

	b.WriteString(m.renderNotification())
	b.WriteString(m.detailHelp(w))
	return b.String()
}

// ── Info tab ────────────────────────────────────────────────────────────

func (m Model) renderInfoTab(c *docker.ContainerInfo, width int) string {
	var b strings.Builder

	if width > 70 {
		b.WriteString(m.renderInfoTwoCol(c, width))
	} else {
		b.WriteString(m.renderInfoSingleCol(c, width))
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
	if !c.Created.IsZero() {
		right.WriteString(renderKV("Created", c.Created.Format("2006-01-02 15:04")))
	}
	if c.Platform != "" {
		right.WriteString(renderKV("Platform", c.Platform))
	}
	if c.RestartCount > 0 {
		right.WriteString(renderKV("Restarts", fmt.Sprintf("%d", c.RestartCount)))
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
	if c.Platform != "" {
		b.WriteString(renderKV("Platform", c.Platform))
	}
	if c.RestartCount > 0 {
		b.WriteString(renderKV("Restarts", fmt.Sprintf("%d", c.RestartCount)))
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
	left.WriteString("  " + progressBar(s.CPUPercent, barW, colorPrimary, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.CPUPercent)) + "\n")

	right.WriteString(sectionHeaderStyle.Width(halfW).Render("  Memory Usage") + "\n")
	right.WriteString("  " + sparklineColored(memH, sparkW, 100, colorCyan) + "\n")
	right.WriteString("  " + progressBar(s.MemPercent, barW, colorCyan, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.MemPercent)) + "\n")
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
	b.WriteString("  " + progressBar(s.CPUPercent, barW, colorPrimary, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.CPUPercent)) + "\n\n")

	b.WriteString(sectionHeaderStyle.Width(width).Render("  Memory Usage") + "\n")
	b.WriteString("  " + sparklineColored(memH, sparkW, 100, colorCyan) + "\n")
	b.WriteString("  " + progressBar(s.MemPercent, barW, colorCyan, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.MemPercent)) + "\n")
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
	if len(m.logLines) == 0 {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No logs available.")
	}

	cleaned := sanitizeOutputText(strings.Join(m.logLines, "\n"))
	if cleaned == "" {
		return lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No logs available.")
	}

	var b strings.Builder
	liveIndicator := ""
	if m.liveLogging {
		liveIndicator = " " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("● LIVE")
	}
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Container Logs"+liveIndicator) + "\n")

	for _, line := range strings.Split(cleaned, "\n") {
		if len(line) > width-4 {
			line = line[:width-4]
		}
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).Render(line) + "\n")
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
		Render("Ctrl+\\ detach | Enter send | x reconnect") + "\n\n")

	out := sanitizeOutputText(m.terminalOutput)
	if out == "" {
		out = "(no terminal output yet)"
	}
	lines := strings.Split(out, "\n")
	show := lines
	maxLines := max(6, m.height-20)
	if len(lines) > maxLines {
		show = lines[len(lines)-maxLines:]
	}
	for _, line := range show {
		if lipgloss.Width(line) > width-4 {
			line = truncate(line, width-4)
		}
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorText).Render(line) + "\n")
	}
	prompt := "  > " + m.terminalInput
	b.WriteString("\n" + inputStyle.Width(width-2).Render(prompt))
	return b.String()
}
