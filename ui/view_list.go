package ui

import (
	"fmt"
	"strings"

	"github.com/akib558/docker-tui/docker"
	"github.com/charmbracelet/lipgloss"
)

// ═══════════════════════════════════════════════════════════════════════
//  LIST VIEW
// ═══════════════════════════════════════════════════════════════════════

func (m Model) viewList() string {
	filtered := m.filteredContainers()
	cols := FitColumns(m.listColumnSpecs(), TableW(m.width))

	s := ListScreen{
		Title:    "Containers",
		Count:    len(m.containers),
		Selected: len(m.selected),
		Cols:     cols,
		Cursor:   m.cursor,
		State:    ScreenReady,
		Help:     m.listHelpKeys(),
	}
	// Same reasoning as the volumes/networks migrations: renderHeader
	// already shows a live typing box while m.filtering is true, so only
	// show the scaffold's own filter bar once filtering is no longer
	// active.
	if !m.filtering {
		s.Filter = m.filterText
	}

	dash := m.dashboardCache
	if dash == "" || m.dashboardCacheW != m.width {
		dash = m.memoizedStatusStrip(m.width)
	}
	s.Extra = append(s.Extra, dash)
	if m.cfg != nil && !m.cfg.HintDismissed {
		s.Extra = append(s.Extra, m.renderGettingStartedHint(m.width))
	}

	switch {
	case m.err != nil && len(m.containers) == 0:
		s.State = ScreenError
		s.Message = fmt.Sprintf("Cannot connect to Docker:\n\n%v\n\nMake sure Docker is running.", m.err)
		return m.renderListScreen(s)
	case m.loading:
		s.State, s.Message = ScreenLoading, "Loading containers..."
		return m.renderListScreen(s)
	case len(m.containers) == 0:
		s.State = ScreenEmpty
		s.Message = "No containers found. Start some containers and press 'r' to refresh."
		return m.renderListScreen(s)
	case len(filtered) == 0:
		s.State = ScreenNoMatch
		s.Message = "No containers match the current filter."
		if m.filterText != "" {
			s.Message = fmt.Sprintf("No containers match filter %q.", m.filterText)
		}
		return m.renderListScreen(s)
	}

	m.fillVisibleListRows(&s, len(filtered), func(i int) ListTableRow {
		ct := filtered[i]
		return ListTableRow{
			Cells:    m.listRowCells(ct, cols),
			Selected: m.selected[ct.ID],
		}
	})
	return m.renderListScreen(s)
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
	return m.memoizedHeader(w)
}

func (m Model) buildHeader(w int) string {
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
		center = selectedMarkStyle.Render(fmt.Sprintf(glyphMarkerSelect+" %d selected", len(m.selected)))
	} else if m.overview != nil {
		dot := lipgloss.NewStyle().Foreground(colorDim).Render(" · ")
		parts := []string{
			lipgloss.NewStyle().Foreground(colorSubtext).Render("docker " + m.overview.ServerVersion),
			lipgloss.NewStyle().Foreground(colorSubtext).Render(fmt.Sprintf("%d images", m.overview.Images)),
		}
		if m.sortMode != sortName {
			sortNames := []string{"name", "state", "cpu", "mem", "image"}
			sortLabel := lipgloss.NewStyle().Foreground(colorSecondary).Render(glyphScroll + " " + sortNames[m.sortMode])
			parts = append(parts, sortLabel)
		}
		center = strings.Join(parts, dot)
	}

	ts := lipgloss.NewStyle().Foreground(colorMuted).Render(nowFunc().Format("15:04:05"))

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
		filterContent += lipgloss.NewStyle().Foreground(colorWarning).Render(glyphBarFill)
	}
	return filterBarStyle.Render(searchIcon + filterContent)
}

// ── Responsive columns ──────────────────────────────────────────────────

// listColumnSpecs declares the container table's columns. FitColumns drops
// them highest-Priority-first as the terminal narrows, which replaces the old
// hand-written width ladder.
func (m Model) listColumnSpecs() []ColumnSpec {
	return []ColumnSpec{
		{Label: "", Width: 3},
		// NAME and STATUS both stay at Priority 0 (never drop) to match the
		// pre-migration width ladder, which showed these two — alongside the
		// state icon — at every width down to the narrowest tier, never
		// hiding either one.
		{Label: "NAME", Percent: 22, Min: 10},
		{Label: "IMAGE", Percent: 22, Min: 10, Priority: 4},
		{Label: "CPU %", Width: 12, Priority: 5},
		{Label: "MEM %", Width: 12, Priority: 5},
		{Label: "STATUS", Percent: 20, Min: 8},
		{Label: "PORTS", Percent: 16, Min: 8, Priority: 6},
		{Label: "CONTAINER ID", Width: 14, Priority: 7},
		{Label: "NET I/O", Width: 16, Priority: 8},
		{Label: "BLOCK I/O", Width: 16, Priority: 8},
	}
}

func (m Model) listRowCells(ct docker.ContainerInfo, cols []ListColumn) []Cell {
	cells := make([]Cell, 0, len(cols))
	for _, c := range cols {
		switch c.Label {
		case "":
			icon := stateGlyph(ct.State)
			if ct.Health != "" {
				icon += healthGlyphPlain(ct.Health)
			}
			cells = append(cells, Cell{Text: glyphPad(icon, c.Width), Width: c.Width, FG: stateFGColor(ct.State)})
		case "NAME":
			cells = append(cells, Cell{Text: truncate(ct.Name, c.Width), Width: c.Width, FG: colorText})
		case "IMAGE":
			cells = append(cells, Cell{Text: truncate(ct.Image, c.Width), Width: c.Width, FG: colorSubtext})
		case "CPU %":
			cells = append(cells, m.statCell(ct.ID, c.Width, func(s *docker.ContainerResourceStats) float64 { return s.CPUPercent }))
		case "MEM %":
			cells = append(cells, m.statCell(ct.ID, c.Width, func(s *docker.ContainerResourceStats) float64 { return s.MemPercent }))
		case "STATUS":
			cells = append(cells, Cell{Text: truncate(ct.Status, c.Width), Width: c.Width, FG: colorSubtext})
		case "PORTS":
			cells = append(cells, Cell{Text: truncate(formatPortsSummary(ct.Ports), c.Width), Width: c.Width, FG: colorSecondary})
		case "CONTAINER ID":
			cells = append(cells, Cell{Text: truncate(ct.ID, c.Width), Width: c.Width, FG: colorDim})
		case "NET I/O":
			cells = append(cells, m.ioCell(ct.ID, c.Width, func(s *docker.ContainerResourceStats) string {
				return formatBytes(s.NetRx) + "/" + formatBytes(s.NetTx)
			}))
		case "BLOCK I/O":
			cells = append(cells, m.ioCell(ct.ID, c.Width, func(s *docker.ContainerResourceStats) string {
				return formatBytes(s.BlockRead) + "/" + formatBytes(s.BlockWrite)
			}))
		}
	}
	return cells
}

// statCell renders a stat as a bar, or a placeholder when stats have not
// arrived for this container yet.
func (m Model) statCell(id string, width int, pick func(*docker.ContainerResourceStats) float64) Cell {
	if s, ok := m.stats[id]; ok {
		return Cell{Width: width, Bar: true, BarPercent: pick(s)}
	}
	return Cell{Width: width, Text: "···", FG: colorDim}
}

// ioCell renders an I/O pair, or "-" when stats have not arrived yet.
func (m Model) ioCell(id string, width int, pick func(*docker.ContainerResourceStats) string) Cell {
	text := "-"
	if s, ok := m.stats[id]; ok {
		text = pick(s)
	}
	return Cell{Text: truncate(text, width), Width: width, FG: colorSubtext}
}
