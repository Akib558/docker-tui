package ui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewEvents() string {
	cols := FitColumns([]ColumnSpec{
		{Label: "TIME", Width: 9},
		{Label: "TYPE", Width: 12, Priority: 2},
		{Label: "ACTION", Width: 14, Priority: 1},
		{Label: "ACTOR", Percent: 100, Min: 12},
		{Label: "CONTAINER ID", Width: 12, Priority: 3},
	}, TableW(m.width))

	s := ListScreen{
		Title:  "Events",
		Count:  len(m.events),
		Cols:   cols,
		Cursor: m.eventsCursor,
		State:  ScreenReady,
		Help:   m.eventsHelpKeys(),
	}
	if len(m.events) == 0 {
		s.State = ScreenEmpty
		s.Message = "Waiting for Docker events..."
		return m.renderListScreen(s)
	}

	s.Rows = make([]ListTableRow, 0, len(m.events))
	for _, ev := range m.events {
		cells := []Cell{{Text: ev.Time.Format("15:04:05"), Width: cols[0].Width, FG: colorMuted}}
		for i, c := range cols[1:] {
			var text string
			var fg lipgloss.Color
			switch c.Label {
			case "TYPE":
				text, fg = ev.Type, eventTypeColor(ev.Type)
			case "ACTION":
				text, fg = ev.Action, eventActionColor(ev.Action)
			case "ACTOR":
				text, fg = ev.Actor, colorText
			case "CONTAINER ID":
				text, fg = ev.ID, colorDim
			}
			cells = append(cells, Cell{Text: truncate(text, cols[i+1].Width), Width: cols[i+1].Width, FG: fg})
		}
		s.Rows = append(s.Rows, ListTableRow{Cells: cells})
	}
	return m.renderListScreen(s)
}

// eventTypeColor tints an event's TYPE column by which kind of Docker object
// it concerns — the pre-migration inline categorization, unchanged.
func eventTypeColor(t string) lipgloss.Color {
	switch t {
	case "container":
		return colorCyan
	case "network":
		return colorSecondary
	case "volume":
		return colorWarning
	default:
		return colorSubtext
	}
}

// eventActionColor tints an event's ACTION column by whether it started or
// ended something — the pre-migration inline categorization, unchanged.
func eventActionColor(action string) lipgloss.Color {
	switch action {
	case "start", "create", "connect":
		return colorSuccess
	case "stop", "die", "destroy", "kill":
		return colorDanger
	default:
		return colorSubtext
	}
}
