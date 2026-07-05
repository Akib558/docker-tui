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

	dash := m.dashboardCache
	if dash == "" || m.dashboardCacheW != w {
		dash = m.renderDashboard(w)
	}
	b.WriteString(dash + "\n")

	if m.cfg != nil && !m.cfg.HintDismissed {
		b.WriteString(m.renderGettingStartedHint(w) + "\n")
	}

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

	frame := m.listFrame(len(filtered))
	visibleRows := frame.BodyRows

	startIdx := 0
	if m.cursor >= visibleRows {
		startIdx = m.cursor - visibleRows + 1
	}
	endIdx := min(startIdx+visibleRows, len(filtered))

	for i := startIdx; i < endIdx; i++ {
		b.WriteString(m.renderTableRow(filtered[i], i == m.cursor, cols, i) + "\n")
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

// appVersion is the docker-tui build version, set from main via SetVersion.
var appVersion = "dev"

// SetVersion records the application version for display in the header.
func SetVersion(v string) {
	if v != "" {
		appVersion = v
	}
}

// headerVersion returns the app version formatted for display (e.g. "v0.1.0").
func headerVersion() string {
	v := appVersion
	if v == "" {
		v = "dev"
	}
	if v != "dev" && !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

func (m Model) renderHeader(w int) string {
	logo := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(glyphLogo + " DOCKER TUI")
	logo += lipgloss.NewStyle().Foreground(colorSubtext).Render(" " + headerVersion())

	var center string
	if m.reconnecting {
		center = lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render(glyphRefresh + " reconnecting...")
	} else if m.filtering {
		center = m.renderFilterBar(m.filterText)
	} else if m.volFiltering && m.view == viewVolumes {
		center = m.renderFilterBar(m.volFilterText)
	} else if m.netFiltering && m.view == viewNetworks {
		center = m.renderFilterBar(m.netFilterText)
	} else if len(m.selected) > 0 {
		center = selectedMarkStyle.Render(fmt.Sprintf("◈ %d selected", len(m.selected)))
	} else if m.overview != nil {
		dot := lipgloss.NewStyle().Foreground(colorDim).Render(" · ")
		parts := []string{
			lipgloss.NewStyle().Foreground(colorSubtext).Render("docker " + m.overview.ServerVersion),
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

	headerBar := lipgloss.NewStyle().Background(colorBgAlt).Width(w).Padding(0, 1).Render(bar)
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", w))
	return headerBar + "\n" + sep + "\n"
}

func (m Model) renderFilterBar(text string) string {
	filterText := text
	if filterText == "" {
		filterText = "type to search..."
	}
	searchIcon := lipgloss.NewStyle().Foreground(colorWarning).Render(glyphSearch + " ")
	filterContent := lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render(filterText)
	if text != "" {
		filterContent += lipgloss.NewStyle().Foreground(colorWarning).Render("█")
	}
	return filterBarStyle.Render(searchIcon + filterContent)
}

// ── Responsive columns ──────────────────────────────────────────────────

type columns struct {
	state, name, image, cpu, mem, status, ports, id, netio, blockio int
	showCPU, showMem, showPorts, showID, showNetIO, showBlockIO     bool
}

func (m Model) calcColumns() columns {
	w := TableW(m.width)
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
	cells := []Cell{
		{Text: "", Width: c.state},
		{Text: "NAME", Width: c.name, FG: colorMuted, Bold: true},
	}
	if c.image > 0 {
		cells = append(cells, Cell{Text: "IMAGE", Width: c.image, FG: colorMuted, Bold: true})
	}
	if c.showCPU {
		cells = append(cells, Cell{Text: "CPU %", Width: c.cpu, FG: colorMuted, Bold: true})
	}
	if c.showMem {
		cells = append(cells, Cell{Text: "MEM %", Width: c.mem, FG: colorMuted, Bold: true})
	}
	cells = append(cells, Cell{Text: "STATUS", Width: c.status, FG: colorMuted, Bold: true})
	if c.showPorts {
		cells = append(cells, Cell{Text: "PORTS", Width: c.ports, FG: colorMuted, Bold: true})
	}
	if c.showID {
		cells = append(cells, Cell{Text: "CONTAINER ID", Width: c.id, FG: colorMuted, Bold: true})
	}
	if c.showNetIO {
		cells = append(cells, Cell{Text: "NET I/O", Width: c.netio, FG: colorMuted, Bold: true})
	}
	if c.showBlockIO {
		cells = append(cells, Cell{Text: "BLOCK I/O", Width: c.blockio, FG: colorMuted, Bold: true})
	}
	row := RenderRow(m.width, cells, RowStyle{Marker: " "})
	return listHeaderStyle.Width(m.width).Render(row)
}

func (m Model) renderTableRow(ct docker.ContainerInfo, isCursor bool, c columns, rowIdx int) string {
	icon := stateIcon(ct.State)
	var stateText string
	if ct.Health != "" {
		stateText = glyphPad(icon+healthIconPlain(ct.Health), c.state)
	} else {
		stateText = glyphPad(icon, c.state)
	}

	cells := []Cell{
		{Text: stateText, Width: c.state, FG: stateFGColor(ct.State)},
		{Text: truncate(ct.Name, c.name), Width: c.name, FG: colorText},
	}
	if c.image > 0 {
		cells = append(cells, Cell{Text: truncate(ct.Image, c.image), Width: c.image, FG: colorSubtext})
	}
	if c.showCPU {
		cell := Cell{Width: c.cpu, FG: colorDim, Text: "···"}
		if s, ok := m.stats[ct.ID]; ok {
			cell = Cell{Width: c.cpu, Bar: true, BarPercent: s.CPUPercent}
		}
		cells = append(cells, cell)
	}
	if c.showMem {
		cell := Cell{Width: c.mem, FG: colorDim, Text: "···"}
		if s, ok := m.stats[ct.ID]; ok {
			cell = Cell{Width: c.mem, Bar: true, BarPercent: s.MemPercent}
		}
		cells = append(cells, cell)
	}
	cells = append(cells, Cell{Text: truncate(ct.Status, c.status), Width: c.status, FG: colorSubtext})
	if c.showPorts {
		p := formatPortsSummary(ct.Ports)
		cells = append(cells, Cell{Text: truncate(p, c.ports), Width: c.ports, FG: colorSecondary})
	}
	if c.showID {
		cells = append(cells, Cell{Text: truncate(ct.ID, c.id), Width: c.id, FG: colorDim})
	}
	if c.showNetIO {
		netStr := "-"
		if s, ok := m.stats[ct.ID]; ok {
			netStr = formatBytes(s.NetRx) + "/" + formatBytes(s.NetTx)
		}
		cells = append(cells, Cell{Text: truncate(netStr, c.netio), Width: c.netio, FG: colorSubtext})
	}
	if c.showBlockIO {
		blockStr := "-"
		if s, ok := m.stats[ct.ID]; ok {
			blockStr = formatBytes(s.BlockRead) + "/" + formatBytes(s.BlockWrite)
		}
		cells = append(cells, Cell{Text: truncate(blockStr, c.blockio), Width: c.blockio, FG: colorSubtext})
	}

	isMultiSel := m.selected[ct.ID]
	var kind ListRowKind
	switch {
	case isCursor && isMultiSel:
		kind = ListRowCursorSelected
	case isCursor:
		kind = ListRowCursor
	case isMultiSel:
		kind = ListRowSelected
	default:
		kind = ListRowNormal
	}
	return renderRowFromKind(m.width, kind, rowIdx, cells)
}
