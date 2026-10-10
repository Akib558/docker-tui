package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewVolumes() string {
	vols := m.filteredVolumes()
	cols := FitColumns([]ColumnSpec{
		{Label: "NAME", Percent: 35, Min: 12},
		{Label: "DRIVER", Width: 10, Priority: 2},
		{Label: "MOUNTPOINT", Percent: 45, Min: 15, Priority: 1},
		{Label: "SCOPE", Width: 8, Priority: 3},
	}, TableW(m.width))

	s := ListScreen{
		Title:    "Volumes",
		Count:    len(m.volumes),
		Selected: len(m.selected),
		Cols:     cols,
		Cursor:   m.volCursor,
		State:    ScreenReady,
		Help:     m.volumesHelpKeys(),
	}
	// renderHeader already shows a live typing box in the header's center
	// while m.volFiltering is true (see its m.volFiltering && m.view ==
	// viewVolumes branch) — showing the scaffold's own filter bar at the
	// same time would duplicate that indicator. Show it only once filtering
	// is no longer active, as a persistent "currently filtering" badge —
	// something the pre-migration view never had.
	if !m.volFiltering {
		s.Filter = m.volFilterText
	}

	switch {
	case m.loading:
		s.State, s.Message = ScreenLoading, "Loading volumes..."
		return m.renderListScreen(s)
	case len(m.volumes) == 0:
		s.State, s.Message = ScreenEmpty, "No volumes found."
		return m.renderListScreen(s)
	case len(vols) == 0:
		s.State, s.Message = ScreenNoMatch, "No volumes match the current filter."
		return m.renderListScreen(s)
	}

	m.fillVisibleListRows(&s, len(vols), func(i int) ListTableRow {
		v := vols[i]
		cells := []Cell{{Text: truncate(v.DisplayName(), cols[0].Width), Width: cols[0].Width, FG: colorText}}
		for j, c := range cols[1:] {
			var text string
			var fg lipgloss.Color
			switch c.Label {
			case "DRIVER":
				text, fg = v.Driver, colorSubtext
			case "MOUNTPOINT":
				text, fg = v.Mountpoint, colorDim
			case "SCOPE":
				text, fg = v.Scope, colorMuted
			}
			cells = append(cells, Cell{Text: truncate(text, cols[j+1].Width), Width: cols[j+1].Width, FG: fg})
		}
		return ListTableRow{Cells: cells, Selected: m.selected[v.Name]}
	})
	return m.renderListScreen(s)
}

func (m Model) volumesHelpKeys() string {
	if m.volFiltering {
		return fmtKeys([]struct{ key, desc string }{
			{"type", "search"},
			{"backspace", "delete"},
			{"enter/esc", "done"},
			{"ctrl+u", "clear"},
		})
	}
	return fmtKeys([]struct{ key, desc string }{
		{"j/k", "nav"},
		{"space", "select"},
		{"d", "remove"},
		{"p", "prune"},
		{"/", "filter"},
		{"?", "help"},
		{"esc", "back"},
	})
}
