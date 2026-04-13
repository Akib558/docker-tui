package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/akib/docker-tui/config"
	"github.com/akib/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

// ── Top-level View ──────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// When a dialog is active, render it as a full-screen overlay
	if m.dialog != dialogNone {
		return m.renderDialogOverlay()
	}

	var content string
	switch m.view {
	case viewList:
		content = m.viewList()
	case viewDetail:
		content = m.viewDetail()
	case viewImages:
		content = m.viewImages()
	case viewEvents:
		content = m.viewEvents()
	}

	// Pad to full terminal height to prevent flicker
	lines := strings.Count(content, "\n") + 1
	if lines < m.height {
		content += strings.Repeat("\n", m.height-lines)
	}
	return content
}

// ── Dialog overlay ───────────────────────────────────────────────────────

func (m Model) renderDialogOverlay() string {
	var d string
	switch m.dialog {
	case dialogConfirm:
		d = m.renderConfirmDialog()
	case dialogTheme:
		d = m.renderThemeDialog()
	case dialogInput:
		d = m.renderInputDialog()
	}
	if d == "" {
		return ""
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, d)
}

func (m Model) renderConfirmDialog() string {
	title := dialogTitleStyle.Render("⚠  Confirm Action")
	msg := lipgloss.NewStyle().Foreground(colorText).Render(m.confirmMsg)
	btns := "\n" + helpKeyStyle.Render("y") + " " + helpDescStyle.Render("confirm") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("n/esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + msg + "\n" + btns
	w := min(64, m.width-8)
	return dialogStyle.Width(w).Render(content)
}

func (m Model) renderThemeDialog() string {
	title := dialogTitleStyle.Render("  Select Theme")
	var lines []string
	for i, t := range config.Themes {
		if i == m.themeCursor {
			line := cursorStyle.Render("▸ ") + listItemSelStyle.Render(" "+t.Name+" ")
			lines = append(lines, line)
		} else {
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(colorText).Render(t.Name))
		}
	}
	help := "\n" + helpKeyStyle.Render("j/k") + " " + helpDescStyle.Render("navigate") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("select") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + strings.Join(lines, "\n") + "\n" + help
	w := min(44, m.width-8)
	return dialogStyle.Width(w).Render(content)
}

func (m Model) renderInputDialog() string {
	title := dialogTitleStyle.Render(m.inputPrompt)
	inputW := min(44, m.width-16)
	cursor := lipgloss.NewStyle().Foreground(colorPrimary).Render("█")
	input := inputStyle.Width(inputW).Render(m.inputText + cursor)
	help := "\n" + helpKeyStyle.Render("enter") + " " + helpDescStyle.Render("submit") +
		"  " + lipgloss.NewStyle().Foreground(colorDim).Render("|") + "  " +
		helpKeyStyle.Render("esc") + " " + helpDescStyle.Render("cancel")
	content := title + "\n\n" + input + "\n" + help
	w := min(54, m.width-8)
	return dialogStyle.Width(w).Render(content)
}

// ═══════════════════════════════════════════════════════════════════════
//  LIST VIEW
// ═══════════════════════════════════════════════════════════════════════

func (m Model) viewList() string {
	var b strings.Builder
	w := m.width
	filtered := m.filteredContainers()

	// ── Header ──────────────────────────────────────────────────────
	b.WriteString(m.renderHeader(w))

	// ── Error ───────────────────────────────────────────────────────
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

	// ── Stat cards + host stats ─────────────────────────────────────
	b.WriteString(m.renderDashboard(w) + "\n")

	// ── Empty state ─────────────────────────────────────────────────
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

	// ── Table ───────────────────────────────────────────────────────
	cols := m.calcColumns()
	b.WriteString(m.renderTableHeader(cols) + "\n")

	// Rows that fit on screen (header=3, dashboard=5, thead=2, help=2, notif=1, pad=1)
	usedLines := 14
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

	// Scroll indicator
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

	// Center content: filter bar, multi-select count, or docker info
	var center string
	if m.filtering {
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
		center = selectedMarkStyle.Render(fmt.Sprintf("◈ %d container(s) selected", len(m.selected)))
	} else if m.overview != nil {
		dot := lipgloss.NewStyle().Foreground(colorDim).Render(" · ")
		parts := []string{
			lipgloss.NewStyle().Foreground(colorSubtext).Render("v" + m.overview.ServerVersion),
			lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("%d images", m.overview.Images)),
			lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("%d CPUs", m.overview.CPUs)),
		}
		center = strings.Join(parts, dot)
	}

	ts := lipgloss.NewStyle().Foreground(colorMuted).Render(time.Now().Format("15:04:05"))

	// Layout: logo [center] timestamp
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

// ── Dashboard (stat cards + host stats) ─────────────────────────────────

func (m Model) renderDashboard(w int) string {
	running, stopped, other := 0, 0, 0
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
	cardW := 16
	if w >= 120 {
		cardW = 18
	} else if w < 60 {
		cardW = 12
	}

	makeCard := func(label, value string, vc lipgloss.Color) string {
		inner := cardW - 6 // account for border+padding
		if inner < 4 {
			inner = 4
		}
		icon := lipgloss.NewStyle().Foreground(vc).Render("▪ ")
		l := statCardLabel.Width(inner).Render(label)
		v := statCardValue.Foreground(vc).Width(inner).Render(value)
		return statCardBorder.Width(cardW).BorderForeground(vc).Render(icon + l + "\n" + "  " + v)
	}

	cards := []string{
		makeCard("TOTAL", fmt.Sprintf("%d", total), colorPrimary),
		makeCard("RUNNING", fmt.Sprintf("%d", running), colorSuccess),
		makeCard("STOPPED", fmt.Sprintf("%d", stopped), colorDanger),
	}
	if other > 0 {
		cards = append(cards, makeCard("OTHER", fmt.Sprintf("%d", other), colorWarning))
	}

	// Host resource card (only on wider terminals)
	if w >= 80 {
		var hostLines []string
		mem := m.systemMem
		if mem.Total > 0 {
			barW := cardW + 2
			if barW < 8 {
				barW = 8
			}
			hostLines = append(hostLines,
				lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("MEM  ")+
					hostMemBar(mem.Percent, barW-5))
			hostLines = append(hostLines,
				lipgloss.NewStyle().Foreground(colorDim).
					Render(fmt.Sprintf("     %s / %s", formatBytes(mem.Used), formatBytes(mem.Total))))
		}
		load := m.systemLoad
		if load.Load1 > 0 {
			hostLines = append(hostLines,
				lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render("LOAD ")+
					lipgloss.NewStyle().Foreground(colorSubtext).
						Render(fmt.Sprintf("%.2f  %.2f  %.2f", load.Load1, load.Load5, load.Load15)))
		}
		if len(hostLines) > 0 {
			hostTitle := lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render("⬡ HOST")
			hostCard := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorCyan).
				Padding(0, 1).
				Render(hostTitle + "\n" + strings.Join(hostLines, "\n"))
			cards = append(cards, hostCard)
		}
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, interleave(cards, "  ")...)
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, row)
}

// ── Responsive columns ──────────────────────────────────────────────────

type columns struct {
	state, name, image, cpu, mem, status, ports, id int
	showCPU, showMem, showPorts, showID             bool
}

func (m Model) calcColumns() columns {
	w := m.width - 6
	c := columns{state: 3}

	switch {
	case w < 55:
		// Tiny: state + name + status
		c.name = max(w*45/100, 10)
		c.status = w - c.state - c.name
	case w < 80:
		// Small: + image
		c.name = w * 25 / 100
		c.image = w * 30 / 100
		c.status = w - c.state - c.name - c.image
	case w < 110:
		// Medium: + CPU + MEM bars
		c.showCPU = true
		c.showMem = true
		c.name = w * 20 / 100
		c.image = w * 22 / 100
		c.cpu = max(w*13/100, 12)
		c.mem = max(w*13/100, 12)
		c.status = w - c.state - c.name - c.image - c.cpu - c.mem
	case w < 145:
		// Wide: + ports
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
		// Extra wide: + ID
		c.showCPU = true
		c.showMem = true
		c.showPorts = true
		c.showID = true
		c.name = w * 15 / 100
		c.image = w * 18 / 100
		c.cpu = max(w*10/100, 12)
		c.mem = max(w*10/100, 12)
		c.status = w * 14 / 100
		c.ports = w * 12 / 100
		c.id = w - c.state - c.name - c.image - c.cpu - c.mem - c.status - c.ports
	}

	return c
}

func (m Model) renderTableHeader(c columns) string {
	sel := "  " // placeholder for cursor/select column (2 chars)
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
	return listHeaderStyle.Width(m.width).Render(parts)
}

func (m Model) renderTableRow(ct docker.ContainerInfo, isCursor bool, c columns) string {
	icon := stateIcon(ct.State)
	stStyle := stateStyle(ct.State)
	isMultiSel := m.selected[ct.ID]

	row := stStyle.Width(c.state).Render(icon) + " " +
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

	// Header
	b.WriteString(m.renderHeader(w))

	// Container identity bar
	icon := stateIcon(c.State)
	stStyle := stateStyle(c.State)
	nameStr := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(c.Name)
	stateStr := stStyle.Render(icon + " " + c.State)
	idStr := lipgloss.NewStyle().Foreground(colorDim).Render(truncate(c.ID, 16))
	imgStr := lipgloss.NewStyle().Foreground(colorSubtext).Render(truncate(c.Image, max(w/3, 20)))
	dot := lipgloss.NewStyle().Foreground(colorDim).Render("  ·  ")
	identity := "  " + nameStr + dot + stateStr + dot + imgStr + dot + idStr
	b.WriteString(identity + "\n\n")

	// Tabs
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

	// Tab content
	contentWidth := w - 8
	if contentWidth < 30 {
		contentWidth = 30
	}

	var tabContent string
	switch m.detailTab {
	case 0:
		tabContent = m.renderInfoTab(c, contentWidth)
	case 1:
		tabContent = m.renderResourcesTab(c, contentWidth)
	case 2:
		tabContent = m.renderEnvTab(c, contentWidth)
	case 3:
		tabContent = m.renderLogsTab(contentWidth)
	case 4:
		tabContent = m.renderTerminalTab(c, contentWidth)
	}

	// Scroll
	lines := strings.Split(tabContent, "\n")
	boxChrome := 15 // header(2) + sep(1) + identity(2) + tabs(1) + sep(1) + box-border(2) + help(1) + padding(5)
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

	// CPU
	left.WriteString(sectionHeaderStyle.Width(halfW).Render("  CPU Usage") + "\n")
	left.WriteString("  " + sparklineColored(cpuH, sparkW, 100, colorPrimary) + "\n")
	left.WriteString("  " + progressBar(s.CPUPercent, barW, colorPrimary, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.CPUPercent)) + "\n")

	// Memory
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

	// Network + Block I/O below
	out += "\n" + m.renderIOStats(s, width)

	return out
}

func (m Model) renderResourcesSingleCol(s *docker.ContainerResourceStats, cpuH, memH []float64, width int) string {
	var b strings.Builder
	sparkW := width - 4
	barW := width - 12

	// CPU
	b.WriteString(sectionHeaderStyle.Width(width).Render("  CPU Usage") + "\n")
	b.WriteString("  " + sparklineColored(cpuH, sparkW, 100, colorPrimary) + "\n")
	b.WriteString("  " + progressBar(s.CPUPercent, barW, colorPrimary, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.CPUPercent)) + "\n\n")

	// Memory
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Memory Usage") + "\n")
	b.WriteString("  " + sparklineColored(memH, sparkW, 100, colorCyan) + "\n")
	b.WriteString("  " + progressBar(s.MemPercent, barW, colorCyan, colorDim) +
		lipgloss.NewStyle().Foreground(colorText).Bold(true).Render(fmt.Sprintf(" %.1f%%", s.MemPercent)) + "\n")
	b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).
		Render(fmt.Sprintf("%s / %s", formatBytes(s.MemUsage), formatBytes(s.MemLimit))) + "\n")

	// I/O
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

	// PIDs
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
	b.WriteString(sectionHeaderStyle.Width(width).Render("  Container Logs (last 50 lines)") + "\n")

	for _, line := range strings.Split(cleaned, "\n") {
		if len(line) > width-4 {
			line = line[:width-4]
		}
		b.WriteString("  " + lipgloss.NewStyle().Foreground(colorSubtext).Render(line) + "\n")
	}
	return b.String()
}

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

func (m Model) viewImages() string {
	var b strings.Builder
	w := m.width

	b.WriteString(m.renderHeader(w))
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Render(fmt.Sprintf("Images  (%d)", len(m.images))) + "\n\n")

	if m.loading {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Loading images...") + "\n")
		b.WriteString(m.imagesHelp(w))
		return b.String()
	}
	if len(m.images) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  No images found.") + "\n")
		b.WriteString(m.imagesHelp(w))
		return b.String()
	}

	// Responsive column widths
	tagW := max(w*35/100, 20)
	idW := 14
	sizeW := 10
	dateW := 16
	usedW := tagW + idW + sizeW + dateW + 8
	if usedW > w-4 {
		tagW = max(w-idW-sizeW-dateW-12, 12)
	}

	// Table header
	hdr := "  " +
		tableHeaderStyle.Width(tagW).Render("TAG") + "  " +
		tableHeaderStyle.Width(idW).Render("IMAGE ID") + "  " +
		tableHeaderStyle.Width(sizeW).Render("SIZE") + "  " +
		tableHeaderStyle.Width(dateW).Render("CREATED")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	// Visible rows
	usedLines := 9
	visibleRows := max(3, m.height-usedLines)
	startIdx := 0
	if m.imgCursor >= visibleRows {
		startIdx = m.imgCursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(m.images))

	for i := startIdx; i < endIdx; i++ {
		img := m.images[i]
		row := lipgloss.NewStyle().Width(tagW).Foreground(colorText).Render(truncate(img.DisplayTag(), tagW-1)) + "  " +
			lipgloss.NewStyle().Width(idW).Foreground(colorDim).Render(truncate(img.ID, idW-1)) + "  " +
			lipgloss.NewStyle().Width(sizeW).Foreground(colorSubtext).Render(formatBytes(uint64(img.Size))) + "  " +
			lipgloss.NewStyle().Width(dateW).Foreground(colorMuted).Render(img.Created.Format("2006-01-02 15:04"))
		if i == m.imgCursor {
			b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(w-4).Render(row) + "\n")
		} else {
			b.WriteString("  " + listItemStyle.Width(w-4).Render(row) + "\n")
		}
	}

	if len(m.images) > visibleRows {
		pct := float64(m.imgCursor) / float64(max(len(m.images)-1, 1)) * 100
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).
			Render(fmt.Sprintf("  ↕ %d/%d (%.0f%%)", m.imgCursor+1, len(m.images), pct)) + "\n")
	}

	b.WriteString("\n" + m.imagesHelp(w))
	return b.String()
}

func (m Model) imagesHelp(w int) string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"p", "pull image"},
		{"d", "remove"},
		{"r", "refresh"},
		{"t", "theme"},
		{"esc", "back"},
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}

func (m Model) viewEvents() string {
	var b strings.Builder
	w := m.width
	b.WriteString(m.renderHeader(w))
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Render(fmt.Sprintf("Events  (%d)", len(m.events))) + "\n\n")

	// Column widths
	timeW := 9
	typeW := 12
	actionW := 14
	actorW := max(w*22/100, 16)
	idW := max(w-timeW-typeW-actionW-actorW-12, 10)

	// Table header
	hdr := "  " +
		tableHeaderStyle.Width(timeW).Render("TIME") + "  " +
		tableHeaderStyle.Width(typeW).Render("TYPE") + "  " +
		tableHeaderStyle.Width(actionW).Render("ACTION") + "  " +
		tableHeaderStyle.Width(actorW).Render("ACTOR") + "  " +
		tableHeaderStyle.Width(idW).Render("CONTAINER ID")
	b.WriteString(listHeaderStyle.Width(w).Render(hdr) + "\n")

	if len(m.events) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Italic(true).
			Render("  Waiting for docker events...") + "\n")
	} else {
		maxRows := max(3, m.height-12)
		start := 0
		if len(m.events) > maxRows {
			start = len(m.events) - maxRows
		}
		for i := start; i < len(m.events); i++ {
			ev := m.events[i]

			// Color-code type
			var typeStr string
			switch ev.Type {
			case "container":
				typeStr = eventTypeContainer.Width(typeW).Render(ev.Type)
			case "network":
				typeStr = eventTypeNetwork.Width(typeW).Render(ev.Type)
			case "volume":
				typeStr = eventTypeVolume.Width(typeW).Render(ev.Type)
			default:
				typeStr = lipgloss.NewStyle().Foreground(colorSubtext).Width(typeW).Render(ev.Type)
			}

			// Color-code action
			var actionStr string
			switch {
			case ev.Action == "start" || ev.Action == "create" || ev.Action == "connect":
				actionStr = eventActionStart.Width(actionW).Render(ev.Action)
			case ev.Action == "stop" || ev.Action == "die" || ev.Action == "destroy" || ev.Action == "kill":
				actionStr = eventActionStop.Width(actionW).Render(ev.Action)
			default:
				actionStr = eventActionOther.Width(actionW).Render(ev.Action)
			}

			row := lipgloss.NewStyle().Foreground(colorMuted).Width(timeW).Render(ev.Time.Format("15:04:05")) + "  " +
				typeStr + "  " +
				actionStr + "  " +
				lipgloss.NewStyle().Foreground(colorText).Width(actorW).Render(truncate(ev.Actor, actorW-1)) + "  " +
				lipgloss.NewStyle().Foreground(colorDim).Width(idW).Render(truncate(ev.ID, idW-1))

			if i == m.cursor {
				b.WriteString(cursorStyle.Render("▸ ") + listItemSelStyle.Width(w-4).Render(row) + "\n")
			} else {
				b.WriteString("  " + listItemStyle.Width(w-4).Render(row) + "\n")
			}
		}
	}

	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"c", "clear"},
		{"esc", "back"},
	}
	b.WriteString("\n" + helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys))))
	return b.String()
}

// ── Help bars ───────────────────────────────────────────────────────────

func (m Model) helpCentered(w int) string {
	var keys []struct{ key, desc string }
	if m.filtering {
		keys = []struct{ key, desc string }{
			{"type", "search"},
			{"backspace", "delete"},
			{"enter/esc", "done"},
			{"ctrl+u", "clear"},
		}
	} else if len(m.selected) > 0 {
		keys = []struct{ key, desc string }{
			{"space", "toggle select"},
			{"a", "select all"},
			{"s", "start/stop"},
			{"R", "restart"},
			{"d", "remove"},
			{"esc/a", "deselect"},
		}
	} else {
		keys = []struct{ key, desc string }{
			{"j/k", "navigate"},
			{"enter", "details"},
			{"space", "select"},
			{"/", "filter"},
			{"s", "start/stop"},
			{"e", "exec shell"},
			{"i", "images"},
			{"v", "events"},
			{"t", "theme"},
			{"q", "quit"},
		}
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}

func (m Model) detailHelp(w int) string {
	var keys []struct{ key, desc string }
	if m.detailTab == 3 {
		live := "start live"
		if m.liveLogging {
			live = "stop live"
		}
		keys = []struct{ key, desc string }{
			{"tab/←/→", "switch tab"},
			{"j/k", "scroll"},
			{"l", live},
			{"s", "start/stop"},
			{"esc", "back"},
		}
	} else if m.detailTab == 4 {
		keys = []struct{ key, desc string }{
			{"type", "input"},
			{"enter", "send"},
			{"ctrl+\\", "detach"},
			{"x", "reconnect"},
			{"esc", "back"},
		}
	} else {
		keys = []struct{ key, desc string }{
			{"tab/←/→", "switch tab"},
			{"j/k", "scroll"},
			{"s", "start/stop"},
			{"R", "restart"},
			{"d", "remove"},
			{"e", "exec shell"},
			{"esc", "back"},
		}
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}

func fmtKeys(keys []struct{ key, desc string }) string {
	sep := " " + lipgloss.NewStyle().Foreground(colorDim).Render("·") + " "
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.key)+" "+helpDescStyle.Render(k.desc))
	}
	return strings.Join(parts, sep)
}

// ── Notification ────────────────────────────────────────────────────────

func (m Model) renderNotification() string {
	if m.notification == "" || time.Since(m.notifyTime) > 4*time.Second {
		return ""
	}
	if m.notifyIsErr {
		return "  " + notifyErrorStyle.Render(m.notification) + "\n"
	}
	return "  " + notifySuccessStyle.Render(m.notification) + "\n"
}

// ── Utilities ───────────────────────────────────────────────────────────

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
			if line[0] == 1 || line[0] == 2 {
				line = line[8:]
			}
		}
		cleaned.WriteString(line + "\n")
	}
	return strings.TrimRight(sanitizeOutputText(cleaned.String()), "\n")
}

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\a]*(\a|\x1b\\)|\x1b[@-_]`)

func sanitizeOutputText(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = ansiEscapeRE.ReplaceAllString(s, "")
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\t':
			return r
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)

	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	emptyRun := 0
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			emptyRun++
			if emptyRun > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		emptyRun = 0
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

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
