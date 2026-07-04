package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

// ═══════════════════════════════════════════════════════════════════════
//  LIST VIEW
// ═══════════════════════════════════════════════════════════════════════

func (m Model) viewList() string {
	var b strings.Builder
	w := m.width
	filtered := m.filteredContainers()

	b.WriteString(m.renderHeader(w))

	if m.err != nil && len(m.containers) == 0 {
		boxW := min(w-4, 70)
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDanger).
			Foreground(colorDanger).
			Padding(1, 2).Width(boxW).
			Render(fmt.Sprintf("Cannot connect to Docker:\n\n%v\n\nMake sure Docker is running.", m.err))
		b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, errBox) + "\n\n")
		b.WriteString(m.helpCentered(w))
		return b.String()
	}

	b.WriteString(m.renderDashboard(w) + "\n")

	if len(m.containers) == 0 {
		empty := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("No containers found. Start some containers and press 'r' to refresh.")
		b.WriteString("\n" + lipgloss.PlaceHorizontal(w, lipgloss.Center, empty) + "\n\n")
		b.WriteString(m.helpCentered(w))
		return b.String()
	}
	if len(filtered) == 0 {
		msg := "No containers match the current filter."
		if m.filterText != "" {
			msg = fmt.Sprintf("No containers match filter %q.", m.filterText)
		}
		empty := lipgloss.NewStyle().Foreground(colorMuted).Italic(true).Render(msg)
		b.WriteString("\n" + lipgloss.PlaceHorizontal(w, lipgloss.Center, empty) + "\n\n")
		b.WriteString(m.helpCentered(w))
		return b.String()
	}

	cols := m.calcColumns()
	b.WriteString(m.renderTableHeader(cols) + "\n")

	dashHeight := m.dashboardHeight(w)
	usedLines := 3 + dashHeight + 1 + 2 + 1 + 1
	visibleRows := m.height - usedLines
	if visibleRows < 3 {
		visibleRows = 3
	}

	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(filtered))

	for i := startIdx; i < endIdx; i++ {
		b.WriteString(m.renderTableRow(filtered[i], i == m.cursor, cols) + "\n")
	}

	if len(filtered) > visibleRows {
		pct := float64(m.cursor) / float64(max(len(filtered)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.cursor+1, len(filtered), pct)) + "\n")
	}

	b.WriteString(m.renderNotification())
	b.WriteString("\n" + m.helpCentered(w))
	return b.String()
}

// ── Header bar ──────────────────────────────────────────────────────────

func (m Model) renderHeader(w int) string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("⬡ DOCKER TUI")

	var center string
	if m.reconnecting {
		center = lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("⟳ reconnecting...")
	} else if m.filtering {
		filterText := m.filterText
		if filterText == "" {
			filterText = "type to search..."
		}
		searchIcon := lipgloss.NewStyle().Foreground(colorWarning).Render("⌕ ")
		filterContent := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render(filterText)
		cursor := lipgloss.NewStyle().Foreground(colorWarning).Render("█")
		if m.filterText != "" {
			filterContent += cursor
		}
		center = filterBarStyle.Render(searchIcon + filterContent)
	} else if len(m.selected) > 0 {
		center = selectedMarkStyle.Render(fmt.Sprintf("◈ %d selected", len(m.selected)))
	} else if m.overview != nil {
		dot := lipgloss.NewStyle().Foreground(colorDim).Render(" · ")
		parts := []string{
			lipgloss.NewStyle().Foreground(colorSubtext).Render("v" + m.overview.ServerVersion),
			lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("%d images", m.overview.Images)),
		}
		if m.sortMode != sortName {
			sortNames := []string{"name", "state", "cpu", "mem", "image"}
			sortLabel := lipgloss.NewStyle().Foreground(colorSecondary).Render("↕ " + sortNames[m.sortMode])
			parts = append(parts, sortLabel)
		}
		center = strings.Join(parts, dot)
	}

	ts := lipgloss.NewStyle().Foreground(colorMuted).Render(time.Now().Format("15:04:05"))

	leftLen := lipgloss.Width(logo)
	midLen := lipgloss.Width(center)
	rightLen := lipgloss.Width(ts)
	totalUsed := leftLen + midLen + rightLen

	var bar string
	if totalUsed+4 <= w && midLen > 0 {
		leftPad := (w - totalUsed) / 2
		rightPad := w - leftLen - leftPad - midLen - rightLen
		if leftPad < 1 {
			leftPad = 1
		}
		if rightPad < 1 {
			rightPad = 1
		}
		bar = logo + strings.Repeat(" ", leftPad) + center + strings.Repeat(" ", rightPad) + ts
	} else {
		gap := w - leftLen - rightLen - 1
		if gap < 1 {
			gap = 1
		}
		bar = logo + strings.Repeat(" ", gap) + ts
	}

	headerBar := lipgloss.NewStyle().Background(colorBgAlt).Width(w).Render(" " + bar + " ")
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", w))
	return headerBar + "\n" + sep + "\n"
}

// ── Dashboard (container list by status + host stats) ───────────────────

func (m Model) renderDashboard(w int) string {
	if len(m.containers) == 0 {
		return ""
	}

	var running, stopped, paused []docker.ContainerInfo
	for _, c := range m.containers {
		switch c.State {
		case "running":
			running = append(running, c)
		case "exited", "dead":
			stopped = append(stopped, c)
		case "paused":
			paused = append(paused, c)
		}
	}

	if w < 60 {
		return m.renderCompactDashboard(w, running, stopped, paused)
	}

	var panels []string

	if len(running) > 0 {
		panels = append(panels, m.renderContainerGroup("RUNNING", running, colorSuccess, w))
	}
	if len(paused) > 0 {
		panels = append(panels, m.renderContainerGroup("PAUSED", paused, colorWarning, w))
	}
	if len(stopped) > 0 {
		panels = append(panels, m.renderContainerGroup("STOPPED", stopped, colorDanger, w))
	}

	hostPanel := m.renderHostPanel(w)
	if hostPanel != "" {
		panels = append(panels, hostPanel)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, interleave(panels, " ")...)
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, row)
}

func (m Model) renderContainerGroup(title string, containers []docker.ContainerInfo, color lipgloss.Color, totalWidth int) string {
	panelW := m.calcGroupPanelWidth(len(containers), totalWidth)
	innerW := panelW - 4

	header := lipgloss.NewStyle().Foreground(color).Bold(true).
		Render(fmt.Sprintf("%s (%d)", title, len(containers)))

	var lines []string
	for _, c := range containers {
		icon := stateIcon(c.State)
		stStyle := stateStyle(c.State)

		// Truncate raw name before ANSI rendering to avoid corrupting escape sequences
		nameAvail := max(innerW-2, 1) // 2 = icon(1) + space(1)
		nameStr := truncate(c.Name, nameAvail)
		nameVisualW := 2 + len(nameStr) // icon + space + name

		line := stStyle.Render(icon) + " " + lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(nameStr)

		// Only append info if there's room (need at least " · X" = 4 chars)
		if nameVisualW+4 < innerW {
			var info []string
			remaining := innerW - nameVisualW - 3 // 3 = " · "
			if c.Image != "" && remaining > 4 {
				img := truncate(c.Image, min(remaining, 20))
				info = append(info, lipgloss.NewStyle().Foreground(colorSubtext).Render(img))
				remaining -= len(img)
			}
			if c.State == "running" && !c.StartedAt.IsZero() {
				uptime := formatUptime(time.Since(c.StartedAt))
				info = append(info, lipgloss.NewStyle().Foreground(colorMuted).Render(uptime))
			}
			if len(info) > 0 {
				line += " " + lipgloss.NewStyle().Foreground(colorDim).Render("·") + " " + strings.Join(info, " ")
			}
		}

		lines = append(lines, line)
	}

	content := header + "\n" + strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1).
		Width(panelW).
		Render(content)
}

func (m Model) renderHostPanel(totalWidth int) string {
	mem := m.systemMem
	load := m.systemLoad
	if mem.Total == 0 && load.Load1 == 0 {
		return ""
	}

	panelW := 22
	if totalWidth >= 140 {
		panelW = 26
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("HOST"))

	if mem.Total > 0 {
		barW := panelW - 8
		if barW < 6 {
			barW = 6
		}
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("MEM ")+
				hostMemBar(mem.Percent, barW))
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorDim).
				Render(fmt.Sprintf("    %s / %s", formatBytes(mem.Used), formatBytes(mem.Total))))
	}
	if load.Load1 > 0 {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("CPU ")+
				lipgloss.NewStyle().Foreground(colorSubtext).
					Render(fmt.Sprintf("%.1f  %.1f", load.Load1, load.Load5)))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan).
		Padding(0, 1).
		Width(panelW).
		Render(strings.Join(lines, "\n"))
}

func (m Model) calcGroupPanelWidth(count int, totalWidth int) int {
	nameW := 12
	if count > 0 {
		maxLen := 0
		for _, c := range m.containers {
			if len(c.Name) > maxLen {
				maxLen = len(c.Name)
			}
		}
		nameW = maxLen + 4
	}
	panelW := nameW + 4
	minW := 18
	maxW := totalWidth / 3
	if panelW < minW {
		panelW = minW
	}
	if panelW > maxW {
		panelW = maxW
	}
	return panelW
}

func (m Model) dashboardHeight(w int) int {
	if len(m.containers) == 0 {
		return 0
	}

	if w < 60 {
		height := 0
		running, stopped, paused := 0, 0, 0
		for _, c := range m.containers {
			switch c.State {
			case "running":
				running++
			case "exited", "dead":
				stopped++
			case "paused":
				paused++
			}
		}
		if running > 0 {
			height += 1 + running // header + containers
		}
		if stopped > 0 {
			height += 1 + stopped
		}
		if paused > 0 {
			height += 1 + paused
		}
		return height
	}

	var running, stopped, paused []docker.ContainerInfo
	for _, c := range m.containers {
		switch c.State {
		case "running":
			running = append(running, c)
		case "exited", "dead":
			stopped = append(stopped, c)
		case "paused":
			paused = append(paused, c)
		}
	}

	maxPanelHeight := 0
	if len(running) > 0 {
		h := m.calcGroupHeight(running, w)
		if h > maxPanelHeight {
			maxPanelHeight = h
		}
	}
	if len(stopped) > 0 {
		h := m.calcGroupHeight(stopped, w)
		if h > maxPanelHeight {
			maxPanelHeight = h
		}
	}
	if len(paused) > 0 {
		h := m.calcGroupHeight(paused, w)
		if h > maxPanelHeight {
			maxPanelHeight = h
		}
	}

	hostH := 0
	if m.systemMem.Total > 0 || m.systemLoad.Load1 > 0 {
		hostH = 5
	}
	if hostH > maxPanelHeight {
		maxPanelHeight = hostH
	}

	return maxPanelHeight
}

func (m Model) calcGroupHeight(containers []docker.ContainerInfo, totalWidth int) int {
	// Each container is on its own line, plus header
	return len(containers) + 1
}

func (m Model) renderCompactDashboard(w int, running, stopped, paused []docker.ContainerInfo) string {
	var b strings.Builder

	if len(running) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).
			Render(fmt.Sprintf("● %d running\n", len(running))))
		for _, c := range running {
			line := lipgloss.NewStyle().Foreground(colorText).Render(c.Name)
			if c.Image != "" {
				line += " " + lipgloss.NewStyle().Foreground(colorDim).Render(truncate(c.Image, 15))
			}
			b.WriteString("  " + line + "\n")
		}
	}

	if len(stopped) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorDanger).Bold(true).
			Render(fmt.Sprintf("○ %d stopped\n", len(stopped))))
		for _, c := range stopped {
			line := lipgloss.NewStyle().Foreground(colorMuted).Render(c.Name)
			if c.Image != "" {
				line += " " + lipgloss.NewStyle().Foreground(colorDim).Render(truncate(c.Image, 15))
			}
			b.WriteString("  " + line + "\n")
		}
	}

	if len(paused) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorWarning).Bold(true).
			Render(fmt.Sprintf("◑ %d paused\n", len(paused))))
		for _, c := range paused {
			line := lipgloss.NewStyle().Foreground(colorSubtext).Render(c.Name)
			if c.Image != "" {
				line += " " + lipgloss.NewStyle().Foreground(colorDim).Render(truncate(c.Image, 15))
			}
			b.WriteString("  " + line + "\n")
		}
	}

	return b.String()
}

// ── Responsive columns ──────────────────────────────────────────────────

type columns struct {
	state, name, image, cpu, mem, status, ports, id, netio, blockio int
	showCPU, showMem, showPorts, showID, showNetIO, showBlockIO     bool
}

func (m Model) calcColumns() columns {
	w := m.width - 6
	c := columns{state: 3}

	switch {
	case w < 55:
		c.name = max(w*45/100, 10)
		c.status = w - c.state - c.name
	case w < 80:
		c.name = w * 25 / 100
		c.image = w * 30 / 100
		c.status = w - c.state - c.name - c.image
	case w < 110:
		c.showCPU = true
		c.showMem = true
		c.name = w * 20 / 100
		c.image = w * 22 / 100
		c.cpu = max(w*13/100, 12)
		c.mem = max(w*13/100, 12)
		c.status = w - c.state - c.name - c.image - c.cpu - c.mem
	case w < 150:
		c.showCPU = true
		c.showMem = true
		c.showPorts = true
		c.name = w * 18 / 100
		c.image = w * 20 / 100
		c.cpu = max(w*12/100, 12)
		c.mem = max(w*12/100, 12)
		c.status = w * 16 / 100
		c.ports = w - c.state - c.name - c.image - c.cpu - c.mem - c.status
	default:
		c.showCPU = true
		c.showMem = true
		c.showPorts = true
		c.showID = true
		c.name = w * 15 / 100
		c.image = w * 17 / 100
		c.cpu = max(w*10/100, 12)
		c.mem = max(w*10/100, 12)
		c.status = w * 14 / 100
		c.ports = w * 12 / 100
		c.id = w - c.state - c.name - c.image - c.cpu - c.mem - c.status - c.ports
	}
	return c
}

func (m Model) renderTableHeader(c columns) string {
	sel := "  "
	parts := sel +
		columnHeaderStyle.Width(c.state).Render("") + " " +
		tableHeaderStyle.Width(c.name).Render("NAME")
	if c.image > 0 {
		parts += " " + tableHeaderStyle.Width(c.image).Render("IMAGE")
	}
	if c.showCPU {
		parts += " " + tableHeaderStyle.Width(c.cpu).Render("CPU %")
	}
	if c.showMem {
		parts += " " + tableHeaderStyle.Width(c.mem).Render("MEM %")
	}
	parts += " " + tableHeaderStyle.Width(c.status).Render("STATUS")
	if c.showPorts {
		parts += " " + tableHeaderStyle.Width(c.ports).Render("PORTS")
	}
	if c.showID {
		parts += " " + tableHeaderStyle.Width(c.id).Render("CONTAINER ID")
	}
	if c.showNetIO {
		parts += " " + tableHeaderStyle.Width(c.netio).Render("NET I/O")
	}
	if c.showBlockIO {
		parts += " " + tableHeaderStyle.Width(c.blockio).Render("BLOCK I/O")
	}
	return listHeaderStyle.Width(m.width).Render(parts)
}

func (m Model) renderTableRow(ct docker.ContainerInfo, isCursor bool, c columns) string {
	icon := stateIcon(ct.State)
	stStyle := stateStyle(ct.State)
	isMultiSel := m.selected[ct.ID]

	stateStr := stStyle.Width(c.state).Render(icon)
	if ct.Health != "" {
		// healthIcon is 1 char; pad to same width as state column for alignment
		stateStr = stStyle.Render(icon) + healthIcon(ct.Health) + strings.Repeat(" ", max(c.state-2, 0))
	}

	row := stateStr + " " +
		lipgloss.NewStyle().Width(c.name).Foreground(colorText).Render(truncate(ct.Name, c.name-1))

	if c.image > 0 {
		row += " " + lipgloss.NewStyle().Width(c.image).Foreground(colorSubtext).Render(truncate(ct.Image, c.image-1))
	}
	if c.showCPU {
		noData := lipgloss.NewStyle().Width(c.cpu).Foreground(colorDim).Render(strings.Repeat("░", max(c.cpu-5, 2)) + " ···")
		cpuStr := noData
		if s, ok := m.stats[ct.ID]; ok {
			cpuStr = lipgloss.NewStyle().Width(c.cpu).Render(miniBar(s.CPUPercent, c.cpu-1))
		}
		row += " " + cpuStr
	}
	if c.showMem {
		noData := lipgloss.NewStyle().Width(c.mem).Foreground(colorDim).Render(strings.Repeat("░", max(c.mem-5, 2)) + " ···")
		memStr := noData
		if s, ok := m.stats[ct.ID]; ok {
			memStr = lipgloss.NewStyle().Width(c.mem).Render(miniBar(s.MemPercent, c.mem-1))
		}
		row += " " + memStr
	}
	row += " " + lipgloss.NewStyle().Width(c.status).Foreground(colorSubtext).Render(truncate(ct.Status, c.status-1))
	if c.showPorts {
		p := formatPortsSummary(ct.Ports)
		row += " " + lipgloss.NewStyle().Width(c.ports).Foreground(colorSecondary).Render(truncate(p, c.ports-1))
	}
	if c.showID {
		row += " " + lipgloss.NewStyle().Width(c.id).Foreground(colorDim).Render(truncate(ct.ID, c.id-1))
	}
	if c.showNetIO {
		netStr := "-"
		if s, ok := m.stats[ct.ID]; ok {
			netStr = formatBytes(s.NetRx) + "/" + formatBytes(s.NetTx)
		}
		row += " " + lipgloss.NewStyle().Width(c.netio).Foreground(colorSubtext).Render(truncate(netStr, c.netio-1))
	}
	if c.showBlockIO {
		blockStr := "-"
		if s, ok := m.stats[ct.ID]; ok {
			blockStr = formatBytes(s.BlockRead) + "/" + formatBytes(s.BlockWrite)
		}
		row += " " + lipgloss.NewStyle().Width(c.blockio).Foreground(colorSubtext).Render(truncate(blockStr, c.blockio-1))
	}

	rowW := m.width - 4
	switch {
	case isCursor && isMultiSel:
		mark := selectedMarkStyle.Render("◉ ")
		return mark + listItemSelStyle.Width(rowW).Render(row)
	case isCursor:
		return cursorStyle.Render("▸ ") + listItemSelStyle.Width(rowW).Render(row)
	case isMultiSel:
		mark := selectedMarkStyle.Render("◈ ")
		return mark + listItemStyle.Background(colorBgSelected).Width(rowW).Render(row)
	default:
		return "  " + listItemStyle.Width(rowW).Render(row)
	}
}
