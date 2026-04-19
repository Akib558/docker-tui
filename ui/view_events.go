package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewEvents() string {
	var b strings.Builder
	w := m.width
	b.WriteString(m.renderHeader(w))
	b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).
		Render(fmt.Sprintf("Events  (%d)", len(m.events))) + "\n\n")

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
		maxRows := max(3, m.height-12)
		start := 0
		if len(m.events) > maxRows {
			start = len(m.events) - maxRows
		}
		for i := start; i < len(m.events); i++ {
			ev := m.events[i]

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
