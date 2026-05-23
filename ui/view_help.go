package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── List help bar ────────────────────────────────────────────────────────

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
			{"L", "logs"},
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
			{"L", "logs"},
			{"i", "images"},
			{"v", "events"},
			{"t", "theme"},
			{"q", "quit"},
		}
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys(keys)))
}

func (m Model) centralLogsHelp(w int) string {
	if m.centralLogFiltering {
		return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys([]struct{ key, desc string }{
			{"type", "filter logs"},
			{"backspace", "delete"},
			{"enter/esc", "done"},
			{"ctrl+u", "clear"},
		})))
	}
	return helpBarStyle.Width(w).Render(lipgloss.PlaceHorizontal(w-2, lipgloss.Center, fmtKeys([]struct{ key, desc string }{
		{"j/k", "scroll"},
		{"pgup/pgdn", "page"},
		{"end", "follow"},
		{"/", "filter"},
		{"esc", "back"},
		{"q", "quit"},
	})))
}

// ── Detail help bar ──────────────────────────────────────────────────────

func (m Model) detailHelp(w int) string {
	var keys []struct{ key, desc string }
	if m.detailTab == tabLogs {
		live := "start live"
		if m.liveLogging {
			live = "stop live"
		}
		keys = []struct{ key, desc string }{
			{"tab/←/→", "switch tab"},
			{"j/k", "scroll"},
			{"pgup/pgdn", "jump"},
			{"home/end", "top/bottom"},
			{"l", live},
			{"s", "start/stop"},
			{"esc", "back"},
		}
	} else if m.detailTab == tabTerminal {
		keys = []struct{ key, desc string }{
			{"↑/↓", "scrollback"},
			{"pgup/pgdn", "jump"},
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

// ── Notification ─────────────────────────────────────────────────────────

func (m Model) renderNotification() string {
	if m.notification == "" || time.Since(m.notifyTime) > 4*time.Second {
		return ""
	}
	if m.notifyIsErr {
		return "  " + notifyErrorStyle.Render(m.notification) + "\n"
	}
	return "  " + notifySuccessStyle.Render(m.notification) + "\n"
}

// ── Key formatting ───────────────────────────────────────────────────────

func fmtKeys(keys []struct{ key, desc string }) string {
	sep := " " + lipgloss.NewStyle().Foreground(colorDim).Render("·") + " "
	var parts []string
	for _, k := range keys {
		parts = append(parts, helpKeyStyle.Render(k.key)+" "+helpDescStyle.Render(k.desc))
	}
	return strings.Join(parts, sep)
}
