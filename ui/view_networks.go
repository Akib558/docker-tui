package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewNetworks() string {
	nets := m.filteredNetworks()
	cols := FitColumns([]ColumnSpec{
		{Label: "NAME", Percent: 40, Min: 12},
		{Label: "DRIVER", Width: 10, Priority: 1},
		{Label: "SCOPE", Width: 8, Priority: 2},
		{Label: "NETWORK ID", Percent: 30, Min: 12, Priority: 3},
	}, TableW(m.width))

	s := ListScreen{
		Title:    "Networks",
		Count:    len(m.networks),
		Selected: len(m.selected),
		Cols:     cols,
		Cursor:   m.netCursor,
		State:    ScreenReady,
		Help:     m.networksHelpKeys(),
	}
	// Same reasoning as the volumes migration: renderHeader already shows a
	// live typing box while m.netFiltering is true, so only show the
	// scaffold's own filter bar once filtering is no longer active.
	if !m.netFiltering {
		s.Filter = m.netFilterText
	}

	switch {
	case m.loading:
		s.State, s.Message = ScreenLoading, "Loading networks..."
		return m.renderListScreen(s)
	case len(m.networks) == 0:
		s.State, s.Message = ScreenEmpty, "No networks found."
		return m.renderListScreen(s)
	case len(nets) == 0:
		s.State, s.Message = ScreenNoMatch, "No networks match the current filter."
		return m.renderListScreen(s)
	}

	m.fillVisibleListRows(&s, len(nets), func(i int) ListTableRow {
		n := nets[i]
		cells := []Cell{{Text: truncate(n.Name, cols[0].Width), Width: cols[0].Width, FG: colorText}}
		for j, c := range cols[1:] {
			var text string
			var fg lipgloss.Color
			switch c.Label {
			case "DRIVER":
				text, fg = n.Driver, colorSubtext
			case "SCOPE":
				text, fg = n.Scope, colorMuted
			case "NETWORK ID":
				text, fg = n.ID, colorDim
			}
			cells = append(cells, Cell{Text: truncate(text, cols[j+1].Width), Width: cols[j+1].Width, FG: fg})
		}
		return ListTableRow{Cells: cells, Selected: m.selected[n.ID]}
	})
	return m.renderListScreen(s)
}

func (m Model) networksHelpKeys() string {
	if m.netFiltering {
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
		{"/", "filter"},
		{"?", "help"},
		{"esc", "back"},
	})
}
