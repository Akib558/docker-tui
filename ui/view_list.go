package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/akib/docker-tui/docker"
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

	cardW := 16
	if w >= 120 {
		cardW = 18
	} else if w < 60 {
		cardW = 12
	}

	makeCard := func(label, value string, vc lipgloss.Color) string {
		inner := cardW - 6
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
	case w < 145:
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
