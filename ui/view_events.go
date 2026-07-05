package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewEvents() string {
	var b strings.Builder
	w := m.width
	b.WriteString(m.renderHeader(w))
	b.WriteString(renderViewTitle("Events", len(m.events), 0))

	timeW := 9
	typeW := 12
	actionW := 14
	actorW := max(w*22/100, 16)
	idW := max(w-timeW-typeW-actionW-actorW-12, 10)

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
		frame := m.eventsFrame()
		maxRows := frame.BodyRows
		start := 0
		if len(m.events) > maxRows {
			start = len(m.events) - maxRows
		}
		for i := start; i < len(m.events); i++ {
			ev := m.events[i]

			typeFG := colorSubtext
			switch ev.Type {
			case "container":
				typeFG = colorCyan
			case "network":
				typeFG = colorSecondary
			case "volume":
				typeFG = colorWarning
			}

			actionFG := colorSubtext
			switch {
			case ev.Action == "start" || ev.Action == "create" || ev.Action == "connect":
				actionFG = colorSuccess
			case ev.Action == "stop" || ev.Action == "die" || ev.Action == "destroy" || ev.Action == "kill":
				actionFG = colorDanger
			}

			cells := []Cell{
				{Text: ev.Time.Format("15:04:05"), Width: timeW, FG: colorMuted},
				{Text: ev.Type, Width: typeW, FG: typeFG},
				{Text: ev.Action, Width: actionW, FG: actionFG},
				{Text: truncate(ev.Actor, actorW), Width: actorW, FG: colorText},
				{Text: truncate(ev.ID, idW), Width: idW, FG: colorDim},
			}
			kind := ListRowNormal
			if i == m.eventsCursor {
				kind = ListRowCursor
			}
			b.WriteString(renderRowFromKind(w, kind, i, cells) + "\n")
		}
	}

	keys := []struct{ key, desc string }{
		{"j/k", "nav"},
		{"c", "clear"},
		{"?", "help"},
		{"esc", "back"},
	}
	b.WriteString("\n" + renderHelpBar(w, fmtKeys(keys)))
	return b.String()
}
